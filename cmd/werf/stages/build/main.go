package build

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/flant/werf/cmd/werf/common"
	"github.com/flant/werf/cmd/werf/common/docker_authorizer"
	"github.com/flant/werf/pkg/build"
	"github.com/flant/werf/pkg/docker"
	"github.com/flant/werf/pkg/image"
	"github.com/flant/werf/pkg/lock"
	"github.com/flant/werf/pkg/logger"
	"github.com/flant/werf/pkg/project_tmp_dir"
	"github.com/flant/werf/pkg/ssh_agent"
	"github.com/flant/werf/pkg/true_git"
	"github.com/flant/werf/pkg/werf"
)

type CmdDataType struct {
	PullUsername string
	PullPassword string

	IntrospectBeforeError bool
	IntrospectAfterError  bool
}

var CmdData CmdDataType
var CommonCmdData common.CmdData

func NewCmd() *cobra.Command {
	return NewCmdWithData(&CmdData, &CommonCmdData)
}

func NewCmdWithData(cmdData *CmdDataType, commonCmdData *common.CmdData) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build [IMAGE_NAME...]",
		Short: "Build stages",
		Long: common.GetLongCommandDescription(`Build stages for images described in the werf.yaml.

The result of build command are built stages pushed into the specified stages repo or locally if --stages=:local.

If one or more IMAGE_NAME parameters specified, werf will build only these images stages from werf.yaml`),
		DisableFlagsInUseLine: true,
		Annotations: map[string]string{
			common.CmdEnvAnno: common.EnvsDescription(common.WerfAnsibleArgs, common.WerfDockerConfig),
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			common.LogVersion()

			return common.LogRunningTime(func() error {
				return runStagesBuild(cmdData, commonCmdData, args)
			})
		},
	}

	common.SetupDir(commonCmdData, cmd)
	common.SetupTmpDir(commonCmdData, cmd)
	common.SetupHomeDir(commonCmdData, cmd)
	common.SetupSSHKey(commonCmdData, cmd)

	common.SetupPullUsername(commonCmdData, cmd)
	common.SetupPullPassword(commonCmdData, cmd)

	cmd.Flags().BoolVarP(&cmdData.IntrospectAfterError, "introspect-error", "", false, "Introspect failed stage in the state, right after running failed assembly instruction")
	cmd.Flags().BoolVarP(&cmdData.IntrospectBeforeError, "introspect-before-error", "", false, "Introspect failed stage in the clean state, before running all assembly instructions of the stage")

	common.SetupStagesRepo(commonCmdData, cmd)
	common.SetupStagesUsername(commonCmdData, cmd)
	common.SetupStagesPassword(commonCmdData, cmd)

	return cmd
}

func runStagesBuild(cmdData *CmdDataType, commonCmdData *common.CmdData, imagesToProcess []string) error {
	if err := werf.Init(*commonCmdData.TmpDir, *commonCmdData.HomeDir); err != nil {
		return fmt.Errorf("initialization error: %s", err)
	}

	if err := lock.Init(); err != nil {
		return err
	}

	if err := true_git.Init(); err != nil {
		return err
	}

	if err := docker.Init(docker_authorizer.GetHomeDockerConfigDir()); err != nil {
		return err
	}

	projectDir, err := common.GetProjectDir(commonCmdData)
	if err != nil {
		return fmt.Errorf("getting project dir failed: %s", err)
	}
	common.LogProjectDir(projectDir)

	werfConfig, err := common.GetWerfConfig(projectDir)
	if err != nil {
		return fmt.Errorf("cannot parse werf config: %s", err)
	}

	projectTmpDir, err := project_tmp_dir.Get()
	if err != nil {
		return fmt.Errorf("getting project tmp dir failed: %s", err)
	}
	defer project_tmp_dir.Release(projectTmpDir)

	dockerAuthorizer, err := docker_authorizer.GetBuildStagesDockerAuthorizer(projectTmpDir, cmdData.PullUsername, cmdData.PullPassword)
	if err != nil {
		return err
	}

	if err := ssh_agent.Init(*commonCmdData.SSHKeys); err != nil {
		return fmt.Errorf("cannot initialize ssh agent: %s", err)
	}
	defer func() {
		err := ssh_agent.Terminate()
		if err != nil {
			logger.LogErrorF("WARNING: ssh agent termination failed: %s\n", err)
		}
	}()

	stagesRepo, err := common.GetStagesRepo(commonCmdData)
	if err != nil {
		return err
	}

	opts := build.BuildStagesOptions{
		ImageBuildOptions: image.BuildOptions{
			IntrospectAfterError:  cmdData.IntrospectAfterError,
			IntrospectBeforeError: cmdData.IntrospectBeforeError,
		},
	}

	c := build.NewConveyor(werfConfig, imagesToProcess, projectDir, projectTmpDir, ssh_agent.SSHAuthSock, dockerAuthorizer)

	if err = c.BuildStages(stagesRepo, opts); err != nil {
		return err
	}

	return nil
}
