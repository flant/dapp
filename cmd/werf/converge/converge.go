package converge

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"helm.sh/helm/v3/pkg/postrender"

	"github.com/werf/werf/pkg/git_repo/gitdata"
	"github.com/werf/werf/pkg/giterminism_manager"

	"github.com/werf/werf/pkg/deploy/helm/chart_extender/helpers"
	"github.com/werf/werf/pkg/deploy/helm/command_helpers"
	"github.com/werf/werf/pkg/deploy/helm/maintenance_helper"

	cmd_helm "helm.sh/helm/v3/cmd/helm"
	helm_v3 "helm.sh/helm/v3/cmd/helm"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli/values"

	"github.com/spf13/cobra"

	"github.com/werf/kubedog/pkg/kube"
	"github.com/werf/logboek"

	"github.com/werf/werf/cmd/werf/common"
	"github.com/werf/werf/pkg/build"
	"github.com/werf/werf/pkg/container_runtime"
	"github.com/werf/werf/pkg/deploy/helm/chart_extender"
	"github.com/werf/werf/pkg/deploy/lock_manager"
	"github.com/werf/werf/pkg/deploy/secrets_manager"
	"github.com/werf/werf/pkg/docker"
	"github.com/werf/werf/pkg/git_repo"
	"github.com/werf/werf/pkg/image"
	"github.com/werf/werf/pkg/ssh_agent"
	"github.com/werf/werf/pkg/storage/lrumeta"
	"github.com/werf/werf/pkg/storage/manager"
	"github.com/werf/werf/pkg/tmp_manager"
	"github.com/werf/werf/pkg/true_git"
	"github.com/werf/werf/pkg/werf"
	"github.com/werf/werf/pkg/werf/global_warnings"
)

var cmdData struct {
	Timeout      int
	AutoRollback bool
}

var commonCmdData common.CmdData

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "converge",
		Short: "Build and push images, then deploy application into Kubernetes",
		Long: common.GetLongCommandDescription(`Build and push images, then deploy application into Kubernetes.

The result of converge command is an application deployed into Kubernetes for current git state. Command will create release and wait until all resources of the release will become ready.

Environment is a required param for the deploy by default, because it is needed to construct Helm Release name and Kubernetes Namespace. Either --env or $WERF_ENV should be specified for command.

Read more info about Helm chart structure, Helm Release name, Kubernetes Namespace and how to change it: https://werf.io/documentation/advanced/helm/releases/naming.html`),
		Example: `# Build and deploy current application state into production environment
werf converge --repo registry.mydomain.com/web --env production`,
		DisableFlagsInUseLine: true,
		Annotations: map[string]string{
			common.CmdEnvAnno: common.EnvsDescription(common.WerfDebugAnsibleArgs, common.WerfSecretKey),
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := common.BackgroundContext()

			defer global_warnings.PrintGlobalWarnings(ctx)

			if err := common.ProcessLogOptions(&commonCmdData); err != nil {
				common.PrintHelp(cmd)
				return err
			}

			common.LogVersion()

			return common.LogRunningTime(func() error {
				return runMain(ctx)
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
	common.SetupSSHKey(&commonCmdData, cmd)

	common.SetupIntrospectAfterError(&commonCmdData, cmd)
	common.SetupIntrospectBeforeError(&commonCmdData, cmd)
	common.SetupIntrospectStage(&commonCmdData, cmd)

	common.SetupSecondaryStagesStorageOptions(&commonCmdData, cmd)
	common.SetupStagesStorageOptions(&commonCmdData, cmd)

	common.SetupDockerConfig(&commonCmdData, cmd, "Command needs granted permissions to read, pull and push images into the specified repo, to pull base images")
	common.SetupInsecureRegistry(&commonCmdData, cmd)
	common.SetupSkipTlsVerifyRegistry(&commonCmdData, cmd)

	common.SetupLogOptions(&commonCmdData, cmd)
	common.SetupLogProjectDir(&commonCmdData, cmd)

	common.SetupSynchronization(&commonCmdData, cmd)

	common.SetupKubeConfig(&commonCmdData, cmd)
	common.SetupKubeConfigBase64(&commonCmdData, cmd)
	common.SetupKubeContext(&commonCmdData, cmd)

	common.SetupStatusProgressPeriod(&commonCmdData, cmd)
	common.SetupHooksStatusProgressPeriod(&commonCmdData, cmd)
	common.SetupReleasesHistoryMax(&commonCmdData, cmd)

	common.SetupRelease(&commonCmdData, cmd)
	common.SetupNamespace(&commonCmdData, cmd)
	common.SetupAddAnnotations(&commonCmdData, cmd)
	common.SetupAddLabels(&commonCmdData, cmd)

	common.SetupSetDockerConfigJsonValue(&commonCmdData, cmd)
	common.SetupSet(&commonCmdData, cmd)
	common.SetupSetString(&commonCmdData, cmd)
	common.SetupSetFile(&commonCmdData, cmd)
	common.SetupValues(&commonCmdData, cmd)
	common.SetupSecretValues(&commonCmdData, cmd)
	common.SetupIgnoreSecretKey(&commonCmdData, cmd)

	common.SetupReportPath(&commonCmdData, cmd)
	common.SetupReportFormat(&commonCmdData, cmd)

	common.SetupVirtualMerge(&commonCmdData, cmd)
	common.SetupVirtualMergeFromCommit(&commonCmdData, cmd)
	common.SetupVirtualMergeIntoCommit(&commonCmdData, cmd)

	common.SetupParallelOptions(&commonCmdData, cmd, common.DefaultBuildParallelTasksLimit)

	common.SetupSkipBuild(&commonCmdData, cmd)

	common.SetupFollow(&commonCmdData, cmd)

	common.SetupDisableAutoHostCleanup(&commonCmdData, cmd)
	common.SetupAllowedDockerStorageVolumeUsage(&commonCmdData, cmd)
	common.SetupAllowedDockerStorageVolumeUsageMargin(&commonCmdData, cmd)
	common.SetupAllowedLocalCacheVolumeUsage(&commonCmdData, cmd)
	common.SetupAllowedLocalCacheVolumeUsageMargin(&commonCmdData, cmd)
	common.SetupDockerServerStoragePath(&commonCmdData, cmd)

	cmd.Flags().IntVarP(&cmdData.Timeout, "timeout", "t", 0, "Resources tracking timeout in seconds")
	cmd.Flags().BoolVarP(&cmdData.AutoRollback, "auto-rollback", "R", common.GetBoolEnvironmentDefaultFalse("WERF_AUTO_ROLLBACK"), "Enable auto rollback of the failed release to the previous deployed release version when current deploy process have failed ($WERF_AUTO_ROLLBACK by default)")
	cmd.Flags().BoolVarP(&cmdData.AutoRollback, "atomic", "", common.GetBoolEnvironmentDefaultFalse("WERF_ATOMIC"), "Enable auto rollback of the failed release to the previous deployed release version when current deploy process have failed ($WERF_ATOMIC by default)")

	return cmd
}

func runMain(ctx context.Context) error {
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

	defer func() {
		if err := common.RunAutoHostCleanup(ctx, &commonCmdData); err != nil {
			logboek.Context(ctx).Error().LogF("Auto host cleanup failed: %s\n", err)
		}
	}()

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

	common.SetupOndemandKubeInitializer(*commonCmdData.KubeContext, *commonCmdData.KubeConfig, *commonCmdData.KubeConfigBase64)
	if err := common.GetOndemandKubeInitializer().Init(ctx); err != nil {
		return err
	}

	if *commonCmdData.Follow {
		logboek.LogOptionalLn()
		return common.FollowGitHead(ctx, &commonCmdData, func(ctx context.Context, headCommitGiterminismManager giterminism_manager.Interface) error {
			return run(ctx, headCommitGiterminismManager)
		})
	} else {
		return run(ctx, giterminismManager)
	}
}

func run(ctx context.Context, giterminismManager giterminism_manager.Interface) error {
	werfConfigPath, werfConfig, err := common.GetRequiredWerfConfig(ctx, &commonCmdData, giterminismManager, common.GetWerfConfigOptions(&commonCmdData, true))
	if err != nil {
		return fmt.Errorf("unable to load werf config: %s", err)
	}

	chartDir, err := common.GetHelmChartDir(werfConfigPath, werfConfig, giterminismManager)
	if err != nil {
		return fmt.Errorf("getting helm chart dir failed: %s", err)
	}

	projectName := werfConfig.Meta.Project

	projectTmpDir, err := tmp_manager.CreateProjectDir(ctx)
	if err != nil {
		return fmt.Errorf("getting project tmp dir failed: %s", err)
	}
	defer tmp_manager.ReleaseProjectDir(projectTmpDir)

	buildOptions, err := common.GetBuildOptions(&commonCmdData, werfConfig)
	if err != nil {
		return err
	}

	var imagesInfoGetters []*image.InfoGetter
	var imagesRepository string
	if len(werfConfig.StapelImages) != 0 || len(werfConfig.ImagesFromDockerfile) != 0 {
		stagesStorageAddress, err := common.GetStagesStorageAddress(&commonCmdData)
		if err != nil {
			return err
		}
		containerRuntime := &container_runtime.LocalDockerServerRuntime{} // TODO
		stagesStorage, err := common.GetStagesStorage(stagesStorageAddress, containerRuntime, &commonCmdData)
		if err != nil {
			return err
		}
		logboek.LogOptionalLn()
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

		imagesRepository = storageManager.StagesStorage.String()

		conveyorOptions, err := common.GetConveyorOptionsWithParallel(&commonCmdData, buildOptions)
		if err != nil {
			return err
		}

		conveyorWithRetry := build.NewConveyorWithRetryWrapper(werfConfig, giterminismManager, nil, giterminismManager.ProjectDir(), projectTmpDir, ssh_agent.SSHAuthSock, containerRuntime, storageManager, storageLockManager, conveyorOptions)
		defer conveyorWithRetry.Terminate()

		if err := conveyorWithRetry.WithRetryBlock(ctx, func(c *build.Conveyor) error {
			if *commonCmdData.SkipBuild {
				if err := c.ShouldBeBuilt(ctx); err != nil {
					return err
				}
			} else {
				if err := c.Build(ctx, buildOptions); err != nil {
					return err
				}
			}

			imagesInfoGetters = c.GetImageInfoGetters()

			return nil
		}); err != nil {
			return err
		}

		logboek.LogOptionalLn()
	}

	secretsManager := secrets_manager.NewSecretsManager(secrets_manager.SecretsManagerOptions{DisableSecretsDecryption: *commonCmdData.IgnoreSecretKey})

	releaseName, err := common.GetHelmRelease(*commonCmdData.Release, *commonCmdData.Environment, werfConfig)
	if err != nil {
		return err
	}

	namespace, err := common.GetKubernetesNamespace(*commonCmdData.Namespace, *commonCmdData.Environment, werfConfig)
	if err != nil {
		return err
	}

	kubeConfigOptions := kube.KubeConfigOptions{
		Context:          *commonCmdData.KubeContext,
		ConfigPath:       *commonCmdData.KubeConfig,
		ConfigDataBase64: *commonCmdData.KubeConfigBase64,
	}

	userExtraAnnotations, err := common.GetUserExtraAnnotations(&commonCmdData)
	if err != nil {
		return err
	}

	userExtraLabels, err := common.GetUserExtraLabels(&commonCmdData)
	if err != nil {
		return err
	}

	var lockManager *lock_manager.LockManager
	if m, err := lock_manager.NewLockManager(namespace); err != nil {
		return fmt.Errorf("unable to create lock manager: %s", err)
	} else {
		lockManager = m
	}

	registryClientHandle, err := common.NewHelmRegistryClientHandle(ctx, &commonCmdData)
	if err != nil {
		return fmt.Errorf("unable to create helm registry client: %s", err)
	}

	wc := chart_extender.NewWerfChart(ctx, giterminismManager, secretsManager, chartDir, cmd_helm.Settings, registryClientHandle, chart_extender.WerfChartOptions{
		SecretValueFiles: common.GetSecretValues(&commonCmdData),
		ExtraAnnotations: userExtraAnnotations,
		ExtraLabels:      userExtraLabels,
	})

	if err := wc.SetEnv(*commonCmdData.Environment); err != nil {
		return err
	}
	if err := wc.SetWerfConfig(werfConfig); err != nil {
		return err
	}

	if vals, err := helpers.GetServiceValues(ctx, werfConfig.Meta.Project, imagesRepository, imagesInfoGetters, helpers.ServiceValuesOptions{
		Namespace:                namespace,
		Env:                      *commonCmdData.Environment,
		SetDockerConfigJsonValue: *commonCmdData.SetDockerConfigJsonValue,
		DockerConfigPath:         *commonCmdData.DockerConfig,
	}); err != nil {
		return fmt.Errorf("error creating service values: %s", err)
	} else {
		wc.SetServiceValues(vals)
	}

	loader.GlobalLoadOptions = &loader.LoadOptions{
		ChartExtender:               wc,
		SubchartExtenderFactoryFunc: func() chart.ChartExtender { return chart_extender.NewWerfSubchart() },
	}

	valueOpts := &values.Options{
		ValueFiles:   common.GetValues(&commonCmdData),
		StringValues: common.GetSetString(&commonCmdData),
		Values:       common.GetSet(&commonCmdData),
		FileValues:   common.GetSetFile(&commonCmdData),
	}

	postRenderer, err := wc.GetPostRenderer()
	if err != nil {
		return err
	}

	actionConfig, err := common.NewActionConfig(ctx, common.GetOndemandKubeInitializer(), namespace, &commonCmdData, registryClientHandle)
	if err != nil {
		return err
	}
	maintenanceHelper := createMaintenanceHelper(ctx, actionConfig, kubeConfigOptions)

	if err := migrateHelm2ToHelm3(ctx, releaseName, namespace, maintenanceHelper, postRenderer, valueOpts, filepath.Join(giterminismManager.ProjectDir(), chartDir), registryClientHandle); err != nil {
		return err
	}

	actionConfig, err = common.NewActionConfig(ctx, common.GetOndemandKubeInitializer(), namespace, &commonCmdData, registryClientHandle)
	if err != nil {
		return err
	}

	helmUpgradeCmd, _ := cmd_helm.NewUpgradeCmd(actionConfig, logboek.OutStream(), cmd_helm.UpgradeCmdOptions{
		PostRenderer:    postRenderer,
		ValueOpts:       valueOpts,
		CreateNamespace: common.NewBool(true),
		Install:         common.NewBool(true),
		Wait:            common.NewBool(true),
		Atomic:          common.NewBool(cmdData.AutoRollback),
		Timeout:         common.NewDuration(time.Duration(cmdData.Timeout) * time.Second),
	})

	return command_helpers.LockReleaseWrapper(ctx, releaseName, lockManager, func() error {
		if err := helmUpgradeCmd.RunE(helmUpgradeCmd, []string{releaseName, filepath.Join(giterminismManager.ProjectDir(), chartDir)}); err != nil {
			return fmt.Errorf("helm upgrade have failed: %s", err)
		}
		return nil
	})
}

func createMaintenanceHelper(ctx context.Context, actionConfig *action.Configuration, kubeConfigOptions kube.KubeConfigOptions) *maintenance_helper.MaintenanceHelper {
	maintenanceOpts := maintenance_helper.MaintenanceHelperOptions{
		KubeConfigOptions: kubeConfigOptions,
	}

	for _, val := range []string{
		os.Getenv("WERF_HELM2_RELEASE_STORAGE_NAMESPACE"),
		os.Getenv("WERF_HELM_RELEASE_STORAGE_NAMESPACE"),
		os.Getenv("TILLER_NAMESPACE"),
	} {
		if val != "" {
			maintenanceOpts.Helm2ReleaseStorageNamespace = val
			break
		}
	}

	for _, val := range []string{
		os.Getenv("WERF_HELM2_RELEASE_STORAGE_TYPE"),
		os.Getenv("WERF_HELM_RELEASE_STORAGE_TYPE"),
	} {
		if val != "" {
			maintenanceOpts.Helm2ReleaseStorageType = val
			break
		}
	}

	return maintenance_helper.NewMaintenanceHelper(actionConfig, maintenanceOpts)
}

func migrateHelm2ToHelm3(ctx context.Context, releaseName, namespace string, maintenanceHelper *maintenance_helper.MaintenanceHelper, postRenderer postrender.PostRenderer, valueOpts *values.Options, fullChartDir string, registryClientHandle *helm_v3.RegistryClientHandle) error {
	if helm2Exists, err := checkHelm2AvailableAndReleaseExists(ctx, releaseName, namespace, maintenanceHelper); err != nil {
		return fmt.Errorf("error checking availability of helm 2 and existance of helm 2 release %q: %s", releaseName, err)
	} else if !helm2Exists {
		return nil
	}

	if helm3Exists, err := checkHelm3ReleaseExists(ctx, releaseName, namespace, maintenanceHelper); err != nil {
		return fmt.Errorf("error checking existance of helm 3 release %q: %s", releaseName, err)
	} else if helm3Exists {
		// helm 2 exists and helm 3 exists
		// migration not needed, but we should warn user that some helm 2 release with the same name exists

		logboek.Context(ctx).Warn().LogF("### Helm 2 and helm 3 release %q exists at the same time ###\n", releaseName)
		logboek.Context(ctx).Warn().LogLn()
		logboek.Context(ctx).Warn().LogF("Found existing helm 2 release %q while there is existing helm 3 release %q in the %q namespace!\n", releaseName, releaseName, namespace)
		logboek.Context(ctx).Warn().LogF("werf will continue deploy process into helm 3 release %q in the %q namespace\n", releaseName, namespace)
		logboek.Context(ctx).Warn().LogF("To disable this warning please remove old helm 2 release %q metadata (fox example using: kubectl -n kube-system delete cm RELEASE_NAME.VERSION)\n", releaseName)
		logboek.Context(ctx).Warn().LogLn()

		return nil
	}

	logboek.Context(ctx).Warn().LogFDetails("Found existing helm 2 release %q, will try to render helm 3 templates and migrate existing release resources to helm 3\n", releaseName)

	logboek.Context(ctx).Default().LogOptionalLn()
	if err := logboek.Context(ctx).LogProcess("Rendering helm 3 templates for the current project state").DoError(func() error {
		actionConfig, err := common.NewActionConfig(ctx, common.GetOndemandKubeInitializer(), namespace, &commonCmdData, registryClientHandle)
		if err != nil {
			return err
		}

		helmTemplateCmd, _ := cmd_helm.NewTemplateCmd(actionConfig, ioutil.Discard, cmd_helm.TemplateCmdOptions{
			PostRenderer: postRenderer,
			ValueOpts:    valueOpts,
			Validate:     common.NewBool(true),
			IncludeCrds:  common.NewBool(true),
			IsUpgrade:    common.NewBool(true),
		})
		return helmTemplateCmd.RunE(helmTemplateCmd, []string{releaseName, fullChartDir})
	}); err != nil {
		return err
	}

	if err := logboek.Context(ctx).Default().LogProcess("Migrating helm 2 release %q to helm 3 in the %q namespace", releaseName, namespace).DoError(func() error {
		if err := maintenance_helper.Migrate2To3(ctx, releaseName, releaseName, namespace, maintenanceHelper); err != nil {
			return fmt.Errorf("error migrating existing helm 2 release %q to helm 3 release %q in the namespace %q: %s", releaseName, releaseName, namespace, err)
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

func checkHelm2AvailableAndReleaseExists(ctx context.Context, releaseName, namespace string, maintenanceHelper *maintenance_helper.MaintenanceHelper) (bool, error) {
	if available, err := maintenanceHelper.CheckHelm2StorageAvailable(ctx); err != nil {
		return false, err
	} else if available {
		foundHelm2Release, err := maintenanceHelper.IsHelm2ReleaseExist(ctx, releaseName)
		if err != nil {
			return false, fmt.Errorf("error checking existance of helm 2 release %q: %s", releaseName, err)
		}

		return foundHelm2Release, nil
	}

	return false, nil
}

func checkHelm3ReleaseExists(ctx context.Context, releaseName, namespace string, maintenanceHelper *maintenance_helper.MaintenanceHelper) (bool, error) {
	foundHelm3Release, err := maintenanceHelper.IsHelm3ReleaseExist(ctx, releaseName)
	if err != nil {
		return false, fmt.Errorf("error checking existance of helm 3 release %q: %s", releaseName, err)
	}

	return foundHelm3Release, nil
}
