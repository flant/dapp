package download

import (
	"fmt"
	"os"

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
	Tag         string
	Destination string
}

var commonCmdData common.CmdData

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "download",
		Short:                 "Download published bundle into directory",
		Long:                  common.GetLongCommandDescription(`Take latest bundle from the specified container registry using specified version tag or version mask and unpack it into provided directory (or into directory named as a resulting chart in the current working directory).`),
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

			return common.LogRunningTime(func() error {
				return runApply()
			})
		},
	}

	common.SetupTmpDir(&commonCmdData, cmd)
	common.SetupHomeDir(&commonCmdData, cmd)

	common.SetupInsecureRegistry(&commonCmdData, cmd)
	common.SetupSkipTlsVerifyRegistry(&commonCmdData, cmd)

	common.SetupStagesStorageOptions(&commonCmdData, cmd) // FIXME

	common.SetupLogOptions(&commonCmdData, cmd)
	common.SetupLogProjectDir(&commonCmdData, cmd)

	defaultTag := os.Getenv("WERF_TAG")
	if defaultTag == "" {
		defaultTag = "latest"
	}
	cmd.Flags().StringVarP(&cmdData.Tag, "tag", "", defaultTag, "Provide exact tag version or semver-based pattern, werf will install or upgrade to the latest version of the specified bundle ($WERF_TAG or latest by default)")
	cmd.Flags().StringVarP(&cmdData.Destination, "destination", "d", os.Getenv("WERF_DESTINATION"), "Download bundle into the provided directory ($WERF_DESTINATION or chart-name by default)")

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
	if err := helm.InitActionConfig(ctx, nil, "", cmd_helm.Settings, registryClientHandle, actionConfig, helm.InitActionConfigOptions{}); err != nil {
		return err
	}

	loader.GlobalLoadOptions = &loader.LoadOptions{}

	// FIXME: support semver-pattern
	bundleRef := fmt.Sprintf("%s:%s", repoAddress, cmdData.Tag)

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

	if err := logboek.Context(ctx).LogProcess("Saving bundle into directory").DoError(func() error {
		if cmd := cmd_helm.NewChartExportCmd(actionConfig, logboek.Context(ctx).OutStream(), cmd_helm.ChartExportCmdOptions{Destination: cmdData.Destination}); cmd != nil {
			if err := cmd.RunE(cmd, []string{bundleRef}); err != nil {
				return fmt.Errorf("error saving bundle to the directory: %s", err)
			}
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}
