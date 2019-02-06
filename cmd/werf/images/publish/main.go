package publish

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/flant/werf/cmd/werf/common"
	"github.com/flant/werf/cmd/werf/common/docker_authorizer"
	"github.com/flant/werf/pkg/build"
	"github.com/flant/werf/pkg/docker"
	"github.com/flant/werf/pkg/lock"
	"github.com/flant/werf/pkg/logger"
	"github.com/flant/werf/pkg/project_tmp_dir"
	"github.com/flant/werf/pkg/ssh_agent"
	"github.com/flant/werf/pkg/true_git"
	"github.com/flant/werf/pkg/werf"
)

type CmdDataType struct {
}

var CmdData CmdDataType
var CommonCmdData common.CmdData

func NewCmd() *cobra.Command {
	return NewCmdWithData(&CommonCmdData)
}

func NewCmdWithData(commonCmdData *common.CmdData) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "publish [IMAGE_NAME...]",
		Short: "Build images and push into images repo",
		Long: common.GetLongCommandDescription(`Build final images and push into images repo.

New docker layer with service info about tagging scheme will be built for each image. Images will be pushed into docker repo with the names IMAGES_REPO/IMAGE_NAME:TAG. See more info about images naming: https://flant.github.io/werf/reference/registry/image_naming.html.

If one or more IMAGE_NAME parameters specified, werf will publish only these images from werf.yaml`),
		DisableFlagsInUseLine: true,
		Annotations: map[string]string{
			common.CmdEnvAnno: common.EnvsDescription(common.WerfDockerConfig, common.WerfInsecureRepo),
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return common.LogRunningTime(func() error {
				common.LogVersion()

				return runImagesPublish(commonCmdData, args)
			})
		},
	}

	common.SetupDir(commonCmdData, cmd)
	common.SetupTmpDir(commonCmdData, cmd)
	common.SetupHomeDir(commonCmdData, cmd)
	common.SetupSSHKey(commonCmdData, cmd)

	common.SetupTag(commonCmdData, cmd)

	common.SetupStagesRepo(commonCmdData, cmd)
	common.SetupStagesUsername(commonCmdData, cmd)
	common.SetupStagesPassword(commonCmdData, cmd)

	common.SetupImagesRepo(commonCmdData, cmd)
	common.SetupImagesUsernameWithUsage(&CommonCmdData, cmd, "Images Docker repo username (granted permission to push images, use WERF_IMAGES_USERNAME environment by default)")
	common.SetupImagesPasswordWithUsage(&CommonCmdData, cmd, "Images Docker repo password (granted permission to push images, use WERF_IMAGES_PASSWORD environment by default)")

	return cmd
}

func runImagesPublish(commonCmdData *common.CmdData, imagesToProcess []string) error {
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

	projectName := werfConfig.Meta.Project

	projectTmpDir, err := project_tmp_dir.Get()
	if err != nil {
		return fmt.Errorf("getting project tmp dir failed: %s", err)
	}
	defer project_tmp_dir.Release(projectTmpDir)

	_, err = common.GetStagesRepo(commonCmdData)
	if err != nil {
		return err
	}

	imagesRepo, err := common.GetImagesRepo(projectName, commonCmdData)
	if err != nil {
		return err
	}

	dockerAuthorizer, err := docker_authorizer.GetImagePublishDockerAuthorizer(projectTmpDir, *CommonCmdData.ImagesUsername, *CommonCmdData.ImagesPassword)
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

	tagOpts, err := common.GetTagOptions(commonCmdData)
	if err != nil {
		return err
	}

	opts := build.PublishImagesOptions{TagOptions: tagOpts}

	c := build.NewConveyor(werfConfig, imagesToProcess, projectDir, projectTmpDir, ssh_agent.SSHAuthSock, dockerAuthorizer)

	if err = c.PublishImages(imagesRepo, opts); err != nil {
		return err
	}

	return nil
}
