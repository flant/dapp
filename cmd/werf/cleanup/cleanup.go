package cleanup

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/werf/logboek"

	"github.com/werf/werf/cmd/werf/common"
	"github.com/werf/werf/pkg/cleaning"
	"github.com/werf/werf/pkg/container_runtime"
	"github.com/werf/werf/pkg/docker"
	"github.com/werf/werf/pkg/git_repo"
	"github.com/werf/werf/pkg/git_repo/gitdata"
	"github.com/werf/werf/pkg/image"
	"github.com/werf/werf/pkg/storage/lrumeta"
	"github.com/werf/werf/pkg/storage/manager"
	"github.com/werf/werf/pkg/tmp_manager"
	"github.com/werf/werf/pkg/true_git"
	"github.com/werf/werf/pkg/werf"
	"github.com/werf/werf/pkg/werf/global_warnings"
)

var commonCmdData common.CmdData

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "cleanup",
		DisableFlagsInUseLine: true,
		Short:                 "Cleanup project images in the container registry",
		Long: common.GetLongCommandDescription(`Safely cleanup unused project images in the container registry.

The command works according to special rules called cleanup policies, which the user defines in werf.yaml (https://werf.io/documentation/reference/werf_yaml.html#configuring-cleanup-policies).

It is safe to run this command periodically (daily is enough) by automated cleanup job in parallel with other werf commands such as build, converge and host cleanup.`),
		Example: `  $ werf cleanup --repo registry.mydomain.com/myproject/werf`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := common.BackgroundContext()

			defer global_warnings.PrintGlobalWarnings(ctx)

			if err := common.ProcessLogOptions(&commonCmdData); err != nil {
				common.PrintHelp(cmd)
				return err
			}
			common.LogVersion()

			return common.LogRunningTime(func() error {
				return runCleanup(ctx)
			})
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

	common.SetupSecondaryStagesStorageOptions(&commonCmdData, cmd)
	common.SetupStagesStorageOptions(&commonCmdData, cmd)
	common.SetupParallelOptions(&commonCmdData, cmd, common.DefaultCleanupParallelTasksLimit)

	common.SetupDockerConfig(&commonCmdData, cmd, "Command needs granted permissions to read, pull and delete images from the specified repo")
	common.SetupInsecureRegistry(&commonCmdData, cmd)
	common.SetupSkipTlsVerifyRegistry(&commonCmdData, cmd)

	common.SetupScanContextNamespaceOnly(&commonCmdData, cmd)
	common.SetupDryRun(&commonCmdData, cmd)

	common.SetupLogOptions(&commonCmdData, cmd)
	common.SetupLogProjectDir(&commonCmdData, cmd)

	common.SetupSynchronization(&commonCmdData, cmd)
	common.SetupKubeConfig(&commonCmdData, cmd)
	common.SetupKubeConfigBase64(&commonCmdData, cmd)
	common.SetupKubeContext(&commonCmdData, cmd)
	common.SetupWithoutKube(&commonCmdData, cmd)
	common.SetupKeepStagesBuiltWithinLastNHours(&commonCmdData, cmd)

	common.SetupDisableAutoHostCleanup(&commonCmdData, cmd)
	common.SetupAllowedDockerStorageVolumeUsage(&commonCmdData, cmd)
	common.SetupAllowedDockerStorageVolumeUsageMargin(&commonCmdData, cmd)
	common.SetupAllowedLocalCacheVolumeUsage(&commonCmdData, cmd)
	common.SetupAllowedLocalCacheVolumeUsageMargin(&commonCmdData, cmd)
	common.SetupDockerServerStoragePath(&commonCmdData, cmd)

	return cmd
}

func runCleanup(ctx context.Context) error {
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

	if err := true_git.Init(true_git.Options{LiveGitOutput: *commonCmdData.LogVerbose || *commonCmdData.LogDebug}); err != nil {
		return err
	}

	if err := image.Init(); err != nil {
		return err
	}

	if err := lrumeta.Init(); err != nil {
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

	defer func() {
		if err := common.RunAutoHostCleanup(ctx, &commonCmdData); err != nil {
			logboek.Context(ctx).Error().LogF("Auto host cleanup failed: %s\n", err)
		}
	}()

	common.SetupOndemandKubeInitializer(*commonCmdData.KubeContext, *commonCmdData.KubeConfig, *commonCmdData.KubeConfigBase64)
	if err := common.GetOndemandKubeInitializer().Init(ctx); err != nil {
		return err
	}

	giterminismManager, err := common.GetGiterminismManager(&commonCmdData)
	if err != nil {
		return err
	}

	common.ProcessLogProjectDir(&commonCmdData, giterminismManager.ProjectDir())

	projectTmpDir, err := tmp_manager.CreateProjectDir(ctx)
	if err != nil {
		return fmt.Errorf("getting project tmp dir failed: %s", err)
	}
	defer tmp_manager.ReleaseProjectDir(projectTmpDir)

	_, werfConfig, err := common.GetRequiredWerfConfig(ctx, &commonCmdData, giterminismManager, common.GetWerfConfigOptions(&commonCmdData, true))
	if err != nil {
		return fmt.Errorf("unable to load werf config: %s", err)
	}

	if !werfConfig.Meta.GitWorktree.GetForceShallowClone() && !werfConfig.Meta.GitWorktree.GetAllowFetchingOriginBranchesAndTags() {
		isShallow, err := giterminismManager.LocalGitRepo().IsShallowClone()
		if err != nil {
			return fmt.Errorf("check shallow clone failed: %s", err)
		}

		if isShallow {
			logboek.Warn().LogLn("Git shallow clone should not be used with images cleanup commands due to incompleteness of the repository history that is extremely essential for proper work.")
			logboek.Warn().LogLn("It is recommended to enable automatic fetch of origin git branches and tags during cleanup process with the gitWorktree.allowFetchOriginBranchesAndTags=true werf.yaml directive (which is enabled by default, http://werf.io/documentation/reference/werf_yaml.html#git-worktree).")
			logboek.Warn().LogLn("If you still want to use shallow clone, add gitWorktree.forceShallowClone=true directive into werf.yaml (http://werf.io/documentation/reference/werf_yaml.html#git-worktree).")

			return fmt.Errorf("git shallow clone is not allowed")
		}
	}

	if werfConfig.Meta.GitWorktree.GetAllowFetchingOriginBranchesAndTags() {
		if err := giterminismManager.LocalGitRepo().SyncWithOrigin(ctx); err != nil {
			return fmt.Errorf("synchronization failed: %s", err)
		}
	}

	projectName := werfConfig.Meta.Project

	containerRuntime := &container_runtime.LocalDockerServerRuntime{} // TODO

	stagesStorageAddress, err := common.GetStagesStorageAddress(&commonCmdData)
	if err != nil {
		logboek.Context(ctx).Default().LogLnDetails(`The "werf cleanup" command is only used to cleaning the container registry. In case you need to clean the runner or the localhost, use the commands of the "werf host" group. 
It is worth noting that auto-cleaning is enabled by default, and manual use is usually not required (if not, we would appreciate feedback and creating an issue https://github.com/werf/werf/issues/new).`)

		return err
	}
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

	if *commonCmdData.Parallel {
		storageManager.StagesStorageManager.EnableParallel(int(*commonCmdData.ParallelTasksLimit))
	}

	imagesNames, err := common.GetManagedImagesNames(ctx, projectName, stagesStorage, werfConfig)
	if err != nil {
		return err
	}
	logboek.Debug().LogF("Managed images names: %v\n", imagesNames)

	kubernetesContextClients, err := common.GetKubernetesContextClients(&commonCmdData)
	if err != nil {
		return fmt.Errorf("unable to get Kubernetes clusters connections: %s", err)
	}

	cleanupOptions := cleaning.CleanupOptions{
		ImageNameList:                           imagesNames,
		LocalGit:                                giterminismManager.LocalGitRepo(),
		KubernetesContextClients:                kubernetesContextClients,
		KubernetesNamespaceRestrictionByContext: common.GetKubernetesNamespaceRestrictionByContext(&commonCmdData, kubernetesContextClients),
		WithoutKube:                             *commonCmdData.WithoutKube,
		GitHistoryBasedCleanupOptions:           werfConfig.Meta.Cleanup,
		KeepStagesBuiltWithinLastNHours:         *commonCmdData.KeepStagesBuiltWithinLastNHours,
		DryRun:                                  *commonCmdData.DryRun,
	}

	logboek.LogOptionalLn()
	if err := cleaning.Cleanup(ctx, projectName, storageManager, storageLockManager, cleanupOptions); err != nil {
		return err
	}

	return nil
}
