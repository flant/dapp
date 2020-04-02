package cmd_factory

import (
	"fmt"
	"path/filepath"

	"github.com/flant/werf/pkg/image"

	"github.com/flant/werf/pkg/stages_manager"

	"github.com/spf13/cobra"

	"github.com/flant/logboek"
	"github.com/flant/shluz"

	"github.com/flant/werf/cmd/werf/common"
	"github.com/flant/werf/pkg/build"
	"github.com/flant/werf/pkg/container_runtime"
	"github.com/flant/werf/pkg/docker"
	"github.com/flant/werf/pkg/logging"
	"github.com/flant/werf/pkg/ssh_agent"
	"github.com/flant/werf/pkg/storage"
	"github.com/flant/werf/pkg/tmp_manager"
	"github.com/flant/werf/pkg/true_git"
	"github.com/flant/werf/pkg/werf"
)

func NewCmdWithData(commonCmdData *common.CmdData) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "publish [IMAGE_NAME...]",
		Short: "Build final images from stages and push into images repo",
		Long: common.GetLongCommandDescription(`Build final images using each specified tag with the tagging strategy and push into images repo.

New docker layer with service info about tagging strategy will be built for each tag of each image from werf.yaml. Images will be pushed into docker repo with the names IMAGES_REPO/IMAGE_NAME:TAG. See more info about publish process: https://werf.io/documentation/reference/publish_process.html.

If one or more IMAGE_NAME parameters specified, werf will publish only these images from werf.yaml.`),
		Example: `  # Publish images into myregistry.mydomain.com/myproject images repo using 'mybranch' tag and git-branch tagging strategy
  $ werf images publish --stages-storage :local --images-repo myregistry.mydomain.com/myproject --tag-git-branch mybranch`,
		DisableFlagsInUseLine: true,
		Annotations:           map[string]string{},
		RunE: func(cmd *cobra.Command, args []string) error {
			return common.LogRunningTime(func() error {
				if err := common.ProcessLogOptions(commonCmdData); err != nil {
					common.PrintHelp(cmd)
					return err
				}
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

	common.SetupStagesStorageOptions(commonCmdData, cmd)
	common.SetupImagesRepoOptions(commonCmdData, cmd)

	common.SetupSynchronization(commonCmdData, cmd)
	common.SetupDockerConfig(commonCmdData, cmd, "Command needs granted permissions to read and pull images from the specified stages storage and push images into images repo")
	common.SetupInsecureRegistry(commonCmdData, cmd)
	common.SetupSkipTlsVerifyRegistry(commonCmdData, cmd)

	common.SetupLogOptions(commonCmdData, cmd)
	common.SetupLogProjectDir(commonCmdData, cmd)

	return cmd
}

func runImagesPublish(commonCmdData *common.CmdData, imagesToProcess []string) error {
	if err := werf.Init(*commonCmdData.TmpDir, *commonCmdData.HomeDir); err != nil {
		return fmt.Errorf("initialization error: %s", err)
	}

	if err := image.Init(); err != nil {
		return err
	}

	if err := shluz.Init(filepath.Join(werf.GetServiceDir(), "locks")); err != nil {
		return err
	}

	if err := true_git.Init(true_git.Options{Out: logboek.GetOutStream(), Err: logboek.GetErrStream(), LiveGitOutput: *commonCmdData.LogVerbose || *commonCmdData.LogDebug}); err != nil {
		return err
	}

	if err := common.DockerRegistryInit(commonCmdData); err != nil {
		return err
	}

	if err := docker.Init(*commonCmdData.DockerConfig, *commonCmdData.LogVerbose, *commonCmdData.LogDebug); err != nil {
		return err
	}

	projectDir, err := common.GetProjectDir(commonCmdData)
	if err != nil {
		return fmt.Errorf("getting project dir failed: %s", err)
	}

	common.ProcessLogProjectDir(commonCmdData, projectDir)

	werfConfig, err := common.GetRequiredWerfConfig(projectDir, true)
	if err != nil {
		return fmt.Errorf("unable to load werf config: %s", err)
	}

	logboek.LogOptionalLn()

	for _, imageToProcess := range imagesToProcess {
		if !werfConfig.HasImage(imageToProcess) {
			return fmt.Errorf("specified image %s is not defined in werf.yaml", logging.ImageLogName(imageToProcess, false))
		}
	}

	projectName := werfConfig.Meta.Project

	projectTmpDir, err := tmp_manager.CreateProjectDir()
	if err != nil {
		return fmt.Errorf("getting project tmp dir failed: %s", err)
	}
	defer tmp_manager.ReleaseProjectDir(projectTmpDir)

	containerRuntime := &container_runtime.LocalDockerServerRuntime{} // TODO

	stagesStorage, err := common.GetStagesStorage(containerRuntime, commonCmdData)
	if err != nil {
		return err
	}

	stagesStorageCache := common.GetStagesStorageCache()
	storageLockManager := &storage.FileLockManager{}
	stagesManager := stages_manager.NewStagesManager(projectName, storageLockManager, stagesStorage, stagesStorageCache)

	_, err = common.GetSynchronization(commonCmdData)
	if err != nil {
		return err
	}

	imagesRepo, err := common.GetImagesRepo(projectName, commonCmdData)
	if err != nil {
		return err
	}

	tagOpts, err := common.GetTagOptions(commonCmdData, common.TagOptionsGetterOptions{})
	if err != nil {
		return err
	}

	if err := ssh_agent.Init(*commonCmdData.SSHKeys); err != nil {
		return fmt.Errorf("cannot initialize ssh agent: %s", err)
	}
	defer func() {
		err := ssh_agent.Terminate()
		if err != nil {
			logboek.LogWarnF("WARNING: ssh agent termination failed: %s\n", err)
		}
	}()

	opts := build.PublishImagesOptions{
		ImagesToPublish: imagesToProcess,
		TagOptions:      tagOpts,
	}

	c := build.NewConveyor(werfConfig, imagesToProcess, projectDir, projectTmpDir, ssh_agent.SSHAuthSock, containerRuntime, stagesManager, imagesRepo, storageLockManager)
	defer c.Terminate()

	if err = c.PublishImages(opts); err != nil {
		return err
	}

	return nil
}
