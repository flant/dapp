package apply

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/werf/werf/pkg/deploy/helm/chart_extender/helpers"
	"github.com/werf/werf/pkg/deploy/helm/command_helpers"

	"github.com/werf/werf/pkg/deploy/lock_manager"

	"github.com/werf/werf/pkg/deploy/helm/chart_extender"

	"helm.sh/helm/v3/pkg/cli/values"

	"github.com/werf/kubedog/pkg/kube"

	uuid "github.com/satori/go.uuid"

	"github.com/werf/werf/pkg/werf/global_warnings"

	"github.com/werf/werf/pkg/deploy/helm"

	cmd_helm "helm.sh/helm/v3/cmd/helm"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"

	"github.com/spf13/cobra"

	"github.com/werf/logboek"

	"github.com/werf/werf/cmd/werf/common"
	"github.com/werf/werf/pkg/werf"
)

var cmdData struct {
	Tag          string
	Timeout      int
	AutoRollback bool
}

var commonCmdData common.CmdData

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "apply",
		Short:                 "Apply bundle into Kubernetes",
		Long:                  common.GetLongCommandDescription(`Take latest bundle from the specified container registry using specified version tag or version mask and apply it as a helm chart into Kubernetes cluster.`),
		DisableFlagsInUseLine: true,
		Annotations: map[string]string{
			common.CmdEnvAnno: common.EnvsDescription(),
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			defer global_warnings.PrintGlobalWarnings(common.BackgroundContext())

			if err := common.ProcessLogOptions(&commonCmdData); err != nil {
				common.PrintHelp(cmd)
				return err
			}

			common.LogVersion()

			return common.LogRunningTime(runApply)
		},
	}

	common.SetupEnvironment(&commonCmdData, cmd)
	common.SetupTmpDir(&commonCmdData, cmd)
	common.SetupHomeDir(&commonCmdData, cmd)

	common.SetupInsecureRegistry(&commonCmdData, cmd)
	common.SetupSkipTlsVerifyRegistry(&commonCmdData, cmd)

	common.SetupStagesStorageOptions(&commonCmdData, cmd) // FIXME

	common.SetupDockerConfig(&commonCmdData, cmd, "Command needs granted permissions to read, pull and push images into the specified repo, to pull base images")
	common.SetupInsecureRegistry(&commonCmdData, cmd)
	common.SetupSkipTlsVerifyRegistry(&commonCmdData, cmd)

	common.SetupLogOptions(&commonCmdData, cmd)
	common.SetupLogProjectDir(&commonCmdData, cmd)

	common.SetupAddAnnotations(&commonCmdData, cmd)
	common.SetupAddLabels(&commonCmdData, cmd)

	common.SetupSetDockerConfigJsonValue(&commonCmdData, cmd)
	common.SetupSet(&commonCmdData, cmd)
	common.SetupSetString(&commonCmdData, cmd)
	common.SetupSetFile(&commonCmdData, cmd)
	common.SetupValues(&commonCmdData, cmd)

	common.SetupKubeConfig(&commonCmdData, cmd)
	common.SetupKubeConfigBase64(&commonCmdData, cmd)
	common.SetupKubeContext(&commonCmdData, cmd)

	common.SetupRelease(&commonCmdData, cmd)
	common.SetupNamespace(&commonCmdData, cmd)
	common.SetupStatusProgressPeriod(&commonCmdData, cmd)
	common.SetupHooksStatusProgressPeriod(&commonCmdData, cmd)
	common.SetupReleasesHistoryMax(&commonCmdData, cmd)

	defaultTag := os.Getenv("WERF_TAG")
	if defaultTag == "" {
		defaultTag = "latest"
	}
	cmd.Flags().StringVarP(&cmdData.Tag, "tag", "", defaultTag, "Provide exact tag version or semver-based pattern, werf will install or upgrade to the latest version of the specified bundle ($WERF_TAG or latest by default)")

	return cmd
}

func runApply() error {
	ctx := common.BackgroundContext()

	if err := werf.Init(*commonCmdData.TmpDir, *commonCmdData.HomeDir); err != nil {
		return fmt.Errorf("initialization error: %s", err)
	}

	if err := common.DockerRegistryInit(ctx, &commonCmdData); err != nil {
		return err
	}

	common.SetupOndemandKubeInitializer(*commonCmdData.KubeContext, *commonCmdData.KubeConfig, *commonCmdData.KubeConfigBase64)
	if err := common.GetOndemandKubeInitializer().Init(ctx); err != nil {
		return err
	}

	userExtraAnnotations, err := common.GetUserExtraAnnotations(&commonCmdData)
	if err != nil {
		return err
	}

	userExtraLabels, err := common.GetUserExtraLabels(&commonCmdData)
	if err != nil {
		return err
	}

	repoAddress, err := common.GetStagesStorageAddress(&commonCmdData)
	if err != nil {
		return err
	}

	cmd_helm.Settings.Debug = *commonCmdData.LogDebug

	registryClientHandle, err := common.NewHelmRegistryClientHandle(ctx, &commonCmdData)
	if err != nil {
		return fmt.Errorf("unable to create helm registry client: %s", err)
	}

	actionConfig := new(action.Configuration)
	if err := helm.InitActionConfig(ctx, common.GetOndemandKubeInitializer(), *commonCmdData.Namespace, cmd_helm.Settings, registryClientHandle, actionConfig, helm.InitActionConfigOptions{
		StatusProgressPeriod:      time.Duration(*commonCmdData.StatusProgressPeriodSeconds) * time.Second,
		HooksStatusProgressPeriod: time.Duration(*commonCmdData.HooksStatusProgressPeriodSeconds) * time.Second,
		KubeConfigOptions: kube.KubeConfigOptions{
			Context:          *commonCmdData.KubeContext,
			ConfigPath:       *commonCmdData.KubeConfig,
			ConfigDataBase64: *commonCmdData.KubeConfigBase64,
		},
		ReleasesHistoryMax: *commonCmdData.ReleasesHistoryMax,
	}); err != nil {
		return err
	}

	loader.GlobalLoadOptions = &loader.LoadOptions{}

	// FIXME: support semver-pattern
	bundleRef := fmt.Sprintf("%s:%s", repoAddress, cmdData.Tag)

	bundleTmpDir := filepath.Join(werf.GetServiceDir(), "tmp", "bundles", uuid.NewV4().String())
	defer os.RemoveAll(bundleTmpDir)

	if err := logboek.Context(ctx).LogProcess("Pulling bundle %q", bundleRef).DoError(func() error {
		if cmd := cmd_helm.NewChartPullCmd(actionConfig, logboek.Context(ctx).OutStream()); cmd != nil {
			if err := cmd.RunE(cmd, []string{bundleRef}); err != nil {
				return fmt.Errorf("error saving bundle to the local chart helm cache: %s", err)
			}
		}
		return nil
	}); err != nil {
		return err
	}

	if err := logboek.Context(ctx).LogProcess("Exporting bundle %q", bundleRef).DoError(func() error {
		if cmd := cmd_helm.NewChartExportCmd(actionConfig, logboek.Context(ctx).OutStream(), cmd_helm.ChartExportCmdOptions{Destination: bundleTmpDir}); cmd != nil {
			if err := cmd.RunE(cmd, []string{bundleRef}); err != nil {
				return fmt.Errorf("error pushing bundle %q: %s", bundleRef, err)
			}
		}
		return nil
	}); err != nil {
		return err
	}

	namespace := common.GetNamespace(&commonCmdData)
	releaseName, err := common.GetRequiredRelease(&commonCmdData)
	if err != nil {
		return err
	}

	var lockManager *lock_manager.LockManager
	if m, err := lock_manager.NewLockManager(namespace); err != nil {
		return fmt.Errorf("unable to create lock manager: %s", err)
	} else {
		lockManager = m
	}

	bundle := chart_extender.NewBundle(ctx, bundleTmpDir, cmd_helm.Settings, registryClientHandle, chart_extender.BundleOptions{})

	postRenderer, err := bundle.GetPostRenderer()
	if err != nil {
		return err
	}

	postRenderer.Add(userExtraAnnotations, userExtraLabels)
	if *commonCmdData.Environment != "" {
		postRenderer.Add(map[string]string{"project.werf.io/env": *commonCmdData.Environment}, nil)
	}

	if vals, err := helpers.GetBundleServiceValues(ctx, helpers.ServiceValuesOptions{
		Env:                      *commonCmdData.Environment,
		Namespace:                namespace,
		SetDockerConfigJsonValue: *commonCmdData.SetDockerConfigJsonValue,
		DockerConfigPath:         *commonCmdData.DockerConfig,
	}); err != nil {
		return fmt.Errorf("error creating service values: %s", err)
	} else {
		bundle.SetServiceValues(vals)
	}

	loader.GlobalLoadOptions = &loader.LoadOptions{
		ChartExtender: bundle,
	}

	helmUpgradeCmd, _ := cmd_helm.NewUpgradeCmd(actionConfig, logboek.Context(ctx).OutStream(), cmd_helm.UpgradeCmdOptions{
		PostRenderer: postRenderer,
		ValueOpts: &values.Options{
			ValueFiles:   common.GetValues(&commonCmdData),
			StringValues: common.GetSetString(&commonCmdData),
			Values:       common.GetSet(&commonCmdData),
			FileValues:   common.GetSetFile(&commonCmdData),
		},
		CreateNamespace: common.NewBool(true),
		Install:         common.NewBool(true),
		Wait:            common.NewBool(true),
		Atomic:          common.NewBool(cmdData.AutoRollback),
		Timeout:         common.NewDuration(time.Duration(cmdData.Timeout) * time.Second),
	})

	return command_helpers.LockReleaseWrapper(ctx, releaseName, lockManager, func() error {
		return helmUpgradeCmd.RunE(helmUpgradeCmd, []string{releaseName, bundle.Dir})
	})
}
