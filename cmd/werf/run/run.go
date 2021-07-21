package run

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/docker/cli/cli"

	"github.com/werf/werf/pkg/git_repo/gitdata"
	"github.com/werf/werf/pkg/giterminism_manager"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/spf13/cobra"

	"github.com/werf/logboek"

	"github.com/werf/werf/cmd/werf/common"
	"github.com/werf/werf/pkg/build"
	"github.com/werf/werf/pkg/container_runtime"
	"github.com/werf/werf/pkg/docker"
	"github.com/werf/werf/pkg/git_repo"
	"github.com/werf/werf/pkg/image"
	"github.com/werf/werf/pkg/logging"
	"github.com/werf/werf/pkg/ssh_agent"
	"github.com/werf/werf/pkg/storage/lrumeta"
	"github.com/werf/werf/pkg/storage/manager"
	"github.com/werf/werf/pkg/tmp_manager"
	"github.com/werf/werf/pkg/true_git"
	"github.com/werf/werf/pkg/werf"
	"github.com/werf/werf/pkg/werf/global_warnings"
)

type cmdDataType struct {
	Shell            bool
	Bash             bool
	RawDockerOptions string

	DockerOptions []string
	DockerCommand []string
	ImageName     string
}

var (
	cmdData       cmdDataType
	commonCmdData common.CmdData
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "run [options] [IMAGE_NAME] [-- COMMAND ARG...]",
		Short:                 "Run container for project image",
		Long:                  common.GetLongCommandDescription(`Run container for specified project image from werf.yaml (build if needed)`),
		DisableFlagsInUseLine: true,
		Example: `  # Run specified image
  $ werf run application

  # Run image with predefined docker run options and command for debug
  $ werf run --shell

  # Run image with specified docker run options and command
  $ werf run --docker-options="-d -p 5000:5000 --restart=always --name registry" -- /app/run.sh

  # Print a resulting docker run command
  $ werf run --shell --dry-run
  docker run -ti --rm image-stage-test:1ffe83860127e68e893b6aece5b0b7619f903f8492a285c6410371c87018c6a0 /bin/sh`,
		Annotations: map[string]string{
			common.DisableOptionsInUseLineAnno: "1",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := common.BackgroundContext()
			defer global_warnings.PrintGlobalWarnings(ctx)

			if err := common.ProcessLogOptions(&commonCmdData); err != nil {
				common.PrintHelp(cmd)
				return err
			}

			if err := processArgs(cmd, args); err != nil {
				common.PrintHelp(cmd)
				return err
			}

			if cmdData.RawDockerOptions != "" {
				cmdData.DockerOptions = strings.Fields(cmdData.RawDockerOptions)
			}

			if cmdData.Shell && cmdData.Bash {
				return fmt.Errorf("cannot use --shell and --bash options at the same time")
			}

			if cmdData.Shell || cmdData.Bash {
				if len(cmdData.DockerOptions) == 0 && len(cmdData.DockerCommand) == 0 {
					cmdData.DockerOptions = []string{"-ti", "--rm"}
					if cmdData.Shell {
						cmdData.DockerCommand = []string{"/bin/sh"}
					}

					if cmdData.Bash {
						cmdData.DockerCommand = []string{"/bin/bash"}
					}
				} else {
					common.PrintHelp(cmd)
					return fmt.Errorf("shell option cannot be used with other docker run arguments")
				}
			}

			return runMain()
		},
	}

	common.SetupDir(&commonCmdData, cmd)
	common.SetupGitWorkTree(&commonCmdData, cmd)
	common.SetupConfigTemplatesDir(&commonCmdData, cmd)
	common.SetupConfigPath(&commonCmdData, cmd)
	common.SetupEnvironment(&commonCmdData, cmd)

	common.SetupGiterminismOptions(&commonCmdData, cmd)

	common.SetupTmpDir(&commonCmdData, cmd)
	common.SetupHomeDir(&commonCmdData, cmd)
	common.SetupSSHKey(&commonCmdData, cmd)

	common.SetupSecondaryStagesStorageOptions(&commonCmdData, cmd)
	common.SetupStagesStorageOptions(&commonCmdData, cmd)

	common.SetupSkipBuild(&commonCmdData, cmd)

	common.SetupFollow(&commonCmdData, cmd)

	common.SetupDockerConfig(&commonCmdData, cmd, "Command needs granted permissions to read and pull images from the specified repo")
	common.SetupInsecureRegistry(&commonCmdData, cmd)
	common.SetupSkipTlsVerifyRegistry(&commonCmdData, cmd)

	common.SetupLogOptions(&commonCmdData, cmd)
	common.SetupLogProjectDir(&commonCmdData, cmd)

	common.SetupSynchronization(&commonCmdData, cmd)
	common.SetupKubeConfig(&commonCmdData, cmd)
	common.SetupKubeConfigBase64(&commonCmdData, cmd)
	common.SetupKubeContext(&commonCmdData, cmd)

	common.SetupDryRun(&commonCmdData, cmd)

	common.SetupVirtualMerge(&commonCmdData, cmd)
	common.SetupVirtualMergeFromCommit(&commonCmdData, cmd)
	common.SetupVirtualMergeIntoCommit(&commonCmdData, cmd)

	cmd.Flags().BoolVarP(&cmdData.Shell, "shell", "", false, "Use predefined docker options and command for debug")
	cmd.Flags().BoolVarP(&cmdData.Bash, "bash", "", false, "Use predefined docker options and command for debug")
	cmd.Flags().StringVarP(&cmdData.RawDockerOptions, "docker-options", "", os.Getenv("WERF_DOCKER_OPTIONS"), "Define docker run options (default $WERF_DOCKER_OPTIONS)")

	return cmd
}

func processArgs(cmd *cobra.Command, args []string) error {
	doubleDashInd := cmd.ArgsLenAtDash()
	doubleDashExist := cmd.ArgsLenAtDash() != -1

	if doubleDashExist {
		if doubleDashInd == len(args) {
			return fmt.Errorf("unsupported position args format")
		}

		switch doubleDashInd {
		case 0:
			cmdData.DockerCommand = args[doubleDashInd:]
		case 1:
			cmdData.ImageName = args[0]
			cmdData.DockerCommand = args[doubleDashInd:]
		default:
			return fmt.Errorf("unsupported position args format")
		}
	} else {
		switch len(args) {
		case 0:
		case 1:
			cmdData.ImageName = args[0]
		default:
			return fmt.Errorf("unsupported position args format")
		}
	}

	return nil
}

func checkDetachDockerOption() error {
	for _, value := range cmdData.DockerOptions {
		if value == "-d" || value == "--detach" {
			return nil
		}
	}

	return fmt.Errorf("the container must be launched in the background (in follow mode): pass -d/--detach with --docker-options option")
}

func getContainerName() string {
	for ind, value := range cmdData.DockerOptions {
		if value == "--name" {
			if ind+1 < len(cmdData.DockerOptions) {
				return cmdData.DockerOptions[ind+1]
			}
		} else if strings.HasPrefix(value, "--name=") {
			return strings.TrimPrefix(value, "--name=")
		}
	}

	return ""
}

func runMain() error {
	ctx := common.BackgroundContext()

	if err := werf.Init(*commonCmdData.TmpDir, *commonCmdData.HomeDir); err != nil {
		return fmt.Errorf("initialization error: %s", err)
	}

	gitDataManager, err := gitdata.GetHostGitDataManager(ctx)
	if err != nil {
		return fmt.Errorf("error getting host git data manager: %s", err)
	}

	if err := git_repo.Init(gitDataManager); err != nil {
		return err
	}

	if err := image.Init(); err != nil {
		return err
	}

	if err := lrumeta.Init(); err != nil {
		return err
	}

	if err := true_git.Init(true_git.Options{LiveGitOutput: *commonCmdData.LogVerbose || *commonCmdData.LogDebug}); err != nil {
		return err
	}

	if err := docker.Init(ctx, *commonCmdData.DockerConfig, *commonCmdData.LogVerbose, *commonCmdData.LogDebug); err != nil {
		return err
	}

	ctxWithDockerCli, err := docker.NewContext(ctx)
	if err != nil {
		return err
	}
	ctx = ctxWithDockerCli

	if err := common.DockerRegistryInit(ctxWithDockerCli, &commonCmdData); err != nil {
		return err
	}

	giterminismManager, err := common.GetGiterminismManager(&commonCmdData)
	if err != nil {
		return err
	}

	common.ProcessLogProjectDir(&commonCmdData, giterminismManager.ProjectDir())

	if err := ssh_agent.Init(ctx, common.GetSSHKey(&commonCmdData)); err != nil {
		return fmt.Errorf("cannot initialize ssh agent: %s", err)
	}
	defer func() {
		err := ssh_agent.Terminate()
		if err != nil {
			logboek.Warn().LogF("WARNING: ssh agent termination failed: %s\n", err)
		}
	}()

	if *commonCmdData.Follow {
		if cmdData.Shell || cmdData.Bash {
			return fmt.Errorf("follow mode does not work with --shell and --bash options")
		}

		if err := checkDetachDockerOption(); err != nil {
			return err
		}

		containerName := getContainerName()
		if containerName == "" {
			return fmt.Errorf("follow mode does not work without specific container name: pass --name=CONTAINER_NAME with --docker-options option")
		}

		return common.FollowGitHead(ctx, &commonCmdData, func(ctx context.Context, headCommitGiterminismManager giterminism_manager.Interface) error {
			if err := safeDockerCliRmFunc(ctx, containerName); err != nil {
				return err
			}

			if err := run(ctx, headCommitGiterminismManager); err != nil {
				return err
			}

			go func() {
				time.Sleep(500 * time.Millisecond)
				fmt.Printf("Attaching to container %s ...\n", containerName)

				resp, err := docker.ContainerAttach(ctx, containerName, types.ContainerAttachOptions{
					Stream: true,
					Stdout: true,
					Stderr: true,
					Logs:   true,
				})
				if err != nil {
					_, _ = fmt.Fprintln(os.Stderr, "WARNING:", err)
				}

				if _, err := stdcopy.StdCopy(os.Stdout, os.Stderr, resp.Reader); err != nil {
					_, _ = fmt.Fprintln(os.Stderr, "WARNING:", err)
				}
			}()

			return nil
		})
	} else {
		if err := run(ctx, giterminismManager); err != nil {
			if statusErr, ok := err.(cli.StatusError); ok {
				common.TerminateWithError(err.Error(), statusErr.StatusCode)
			}

			return err
		}

		return nil
	}
}

func run(ctx context.Context, giterminismManager giterminism_manager.Interface) error {
	_, werfConfig, err := common.GetRequiredWerfConfig(ctx, &commonCmdData, giterminismManager, common.GetWerfConfigOptions(&commonCmdData, false))
	if err != nil {
		return fmt.Errorf("unable to load werf config: %s", err)
	}

	projectName := werfConfig.Meta.Project

	projectTmpDir, err := tmp_manager.CreateProjectDir(ctx)
	if err != nil {
		return fmt.Errorf("getting project tmp dir failed: %s", err)
	}
	defer tmp_manager.ReleaseProjectDir(projectTmpDir)

	imageName := cmdData.ImageName
	if imageName == "" && len(werfConfig.GetAllImages()) == 1 {
		imageName = werfConfig.GetAllImages()[0].GetName()
	}

	if !werfConfig.HasImage(imageName) {
		return fmt.Errorf("image %q is not defined in werf.yaml", logging.ImageLogName(imageName, false))
	}

	stagesStorageAddress := common.GetOptionalStagesStorageAddress(&commonCmdData)
	containerRuntime := &container_runtime.LocalDockerServerRuntime{} // TODO
	stagesStorage, err := common.GetStagesStorage(stagesStorageAddress, containerRuntime, &commonCmdData)
	if err != nil {
		return err
	}
	synchronization, err := common.GetSynchronization(ctx, &commonCmdData, projectName, stagesStorage)
	if err != nil {
		return err
	}
	stagesStorageCache, err := common.GetStagesStorageCache(synchronization)
	if err != nil {
		return err
	}
	storageLockManager, err := common.GetStorageLockManager(ctx, synchronization)
	if err != nil {
		return err
	}
	secondaryStagesStorageList, err := common.GetSecondaryStagesStorageList(stagesStorage, containerRuntime, &commonCmdData)
	if err != nil {
		return err
	}

	storageManager := manager.NewStorageManager(projectName, stagesStorage, secondaryStagesStorageList, storageLockManager, stagesStorageCache)

	logboek.Context(ctx).Info().LogOptionalLn()

	conveyorWithRetry := build.NewConveyorWithRetryWrapper(werfConfig, giterminismManager, []string{imageName}, giterminismManager.ProjectDir(), projectTmpDir, ssh_agent.SSHAuthSock, containerRuntime, storageManager, storageLockManager, common.GetConveyorOptions(&commonCmdData))
	defer conveyorWithRetry.Terminate()

	var dockerImageName string
	if err := conveyorWithRetry.WithRetryBlock(ctx, func(c *build.Conveyor) error {
		if *commonCmdData.SkipBuild {
			if err := c.ShouldBeBuilt(ctx); err != nil {
				return err
			}
		} else {
			if err := c.Build(ctx, build.BuildOptions{}); err != nil {
				return err
			}
		}

		if err := c.FetchLastImageStage(ctx, imageName); err != nil {
			return err
		}

		dockerImageName = c.GetImageNameForLastImageStage(imageName)
		return nil
	}); err != nil {
		return err
	}

	var dockerRunArgs []string
	dockerRunArgs = append(dockerRunArgs, cmdData.DockerOptions...)
	dockerRunArgs = append(dockerRunArgs, dockerImageName)
	dockerRunArgs = append(dockerRunArgs, cmdData.DockerCommand...)

	if *commonCmdData.DryRun {
		fmt.Printf("docker run %s\n", strings.Join(dockerRunArgs, " "))
		return nil
	} else {
		return logboek.Streams().DoErrorWithoutProxyStreamDataFormatting(func() error {
			return common.WithoutTerminationSignalsTrap(func() error {
				return docker.CliRun_LiveOutput(ctx, dockerRunArgs...)
			})
		})
	}
}

func safeDockerCliRmFunc(ctx context.Context, containerName string) error {
	if exist, err := docker.ContainerExist(ctx, containerName); err != nil {
		return fmt.Errorf("unable to check container %s existence: %s", containerName, err)
	} else if exist {
		logboek.Context(ctx).LogF("Removing container %s ...\n", containerName)
		if err := docker.CliRm(ctx, "-f", containerName); err != nil {
			return fmt.Errorf("unable to remove container %s: %s", containerName, err)
		}
	}

	return nil
}
