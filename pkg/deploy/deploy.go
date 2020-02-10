package deploy

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/flant/werf/pkg/util/secretvalues"

	"github.com/ghodss/yaml"

	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/proto/hapi/chart"

	"github.com/flant/kubedog/pkg/kube"
	"github.com/flant/logboek"

	"github.com/flant/werf/pkg/config"
	"github.com/flant/werf/pkg/deploy/helm"
	"github.com/flant/werf/pkg/deploy/werf_chart"
	"github.com/flant/werf/pkg/tag_strategy"
)

type DeployOptions struct {
	Values               []string
	SecretValues         []string
	Set                  []string
	SetString            []string
	Timeout              time.Duration
	Env                  string
	UserExtraAnnotations map[string]string
	UserExtraLabels      map[string]string
	IgnoreSecretKey      bool
	ThreeWayMergeMode    helm.ThreeWayMergeModeType
}

type ImagesRepoManager interface {
	ImagesRepo() string
	ImageRepo(imageName string) string
	ImageRepoWithTag(imageName, tag string) string
}

func Deploy(projectDir string, imagesRepoManager ImagesRepoManager, release, namespace, tag string, tagStrategy tag_strategy.TagStrategy, werfConfig *config.WerfConfig, helmReleaseStorageNamespace, helmReleaseStorageType string, opts DeployOptions) error {
	var logBlockErr error
	var werfChart *werf_chart.WerfChart

	logboek.LogBlock("Deploy options", logboek.LogBlockOptions{}, func() {
		if kube.Context != "" {
			logboek.LogF("Using kube context: %s\n", kube.Context)
		}
		logboek.LogF("Using helm release storage namespace: %s\n", helmReleaseStorageNamespace)
		logboek.LogF("Using helm release storage type: %s\n", helmReleaseStorageType)
		logboek.LogF("Using helm release name: %s\n", release)
		logboek.LogF("Using Kubernetes namespace: %s\n", namespace)

		images := GetImagesInfoGetters(werfConfig.StapelImages, werfConfig.ImagesFromDockerfile, imagesRepoManager, tag, false)

		m, err := GetSafeSecretManager(projectDir, opts.SecretValues, opts.IgnoreSecretKey)
		if err != nil {
			logBlockErr = err
			return
		}

		serviceValues, err := GetServiceValues(werfConfig.Meta.Project, imagesRepoManager, namespace, tag, tagStrategy, images, ServiceValuesOptions{Env: opts.Env})
		if err != nil {
			logBlockErr = fmt.Errorf("error creating service values: %s", err)
			return
		}

		serviceValuesRaw, _ := yaml.Marshal(serviceValues)
		logboek.LogLn()
		logboek.LogLn("Using service values:")
		logboek.LogLn(logboek.FitText(string(serviceValuesRaw), logboek.FitTextOptions{ExtraIndentWidth: 2}))

		projectChartDir := filepath.Join(projectDir, werf_chart.ProjectHelmChartDirName)
		werfChart, err = PrepareWerfChart(werfConfig.Meta.Project, projectChartDir, opts.Env, m, opts.SecretValues, serviceValues)
		if err != nil {
			logBlockErr = err
			return
		}
		helm.SetReleaseLogSecretValuesToMask(werfChart.SecretValuesToMask)

		werfChart.MergeExtraAnnotations(opts.UserExtraAnnotations)
		werfChart.MergeExtraLabels(opts.UserExtraLabels)
		werfChart.LogExtraAnnotations()
		werfChart.LogExtraLabels()
	})
	logboek.LogOptionalLn()

	if logBlockErr != nil {
		return logBlockErr
	}

	helm.WerfTemplateEngine.InitWerfEngineExtraTemplatesFunctions(werfChart.DecodedSecretFilesData)
	PatchLoadChartfile(werfChart.Name)

	err := helm.WerfTemplateEngineWithExtraAnnotationsAndLabels(werfChart.ExtraAnnotations, werfChart.ExtraLabels, func() error {
		return werfChart.Deploy(release, namespace, helm.ChartOptions{
			Timeout: opts.Timeout,
			ChartValuesOptions: helm.ChartValuesOptions{
				Set:       opts.Set,
				SetString: opts.SetString,
				Values:    opts.Values,
			},
			ThreeWayMergeMode: opts.ThreeWayMergeMode,
		})
	})

	if err != nil {
		return fmt.Errorf("%s", secretvalues.MaskSecretValuesInString(werfChart.SecretValuesToMask, err.Error()))
	}

	return nil
}

func PatchLoadChartfile(chartName string) {
	boundedFunc := helm.LoadChartfileFunc
	var mu sync.Mutex
	helm.LoadChartfileFunc = func(chartPath string) (*chart.Chart, error) {
		mu.Lock()
		defer mu.Unlock()

		var c *chart.Chart

		if err := chartutil.WithSkipChartYamlFileValidation(true, func() error {
			var err error
			if c, err = boundedFunc(chartPath); err != nil {
				return err
			}

			return nil
		}); err != nil {
			return nil, err
		}

		c.Metadata = &chart.Metadata{
			Name:    chartName,
			Version: "0.1.0",
			Engine:  helm.WerfTemplateEngineName,
		}
		return c, nil
	}
}
