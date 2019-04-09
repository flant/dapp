package deploy

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/flant/logboek"
	"github.com/flant/werf/pkg/deploy/helm"
	"github.com/flant/werf/pkg/tag_strategy"

	"github.com/flant/werf/pkg/config"
)

type LintOptions struct {
	Values       []string
	SecretValues []string
	Set          []string
	SetString    []string
	Env          string
}

func RunLint(projectDir string, werfConfig *config.WerfConfig, opts LintOptions) error {
	if debug() {
		fmt.Fprintf(logboek.GetOutStream(), "Lint options: %#v\n", opts)
	}

	m, err := GetSafeSecretManager(projectDir, opts.SecretValues)
	if err != nil {
		return fmt.Errorf("cannot get project secret: %s", err)
	}

	imagesRepo := "REPO"
	tag := "GIT_BRANCH"
	tagStrategy := tag_strategy.GitBranch
	namespace := "NAMESPACE"

	images := GetImagesInfoGetters(werfConfig.Images, imagesRepo, tag, true)

	serviceValues, err := GetServiceValues(werfConfig.Meta.Project, imagesRepo, namespace, tag, tagStrategy, images, ServiceValuesOptions{Env: opts.Env})
	if err != nil {
		return fmt.Errorf("error creating service values: %s", err)
	}

	werfChart, err := PrepareWerfChart(GetTmpWerfChartPath(werfConfig.Meta.Project), werfConfig.Meta.Project, projectDir, m, opts.SecretValues, serviceValues)
	if err != nil {
		return err
	}
	defer ReleaseTmpWerfChart(werfChart.ChartDir)

	out := &bytes.Buffer{}
	if err := helm.Lint(
		out,
		werfChart.ChartDir,
		namespace,
		append(werfChart.Values, opts.Values...),
		append(werfChart.Set, opts.Set...),
		append(werfChart.SetString, opts.SetString...),
		helm.LintOptions{Strict: true},
	); err != nil {
		replaceOld := fmt.Sprintf("%s/", werfChart.Name)
		replaceNew := fmt.Sprintf("%s/", ".helm")
		errMsg := strings.Replace(err.Error(), replaceOld, replaceNew, -1)
		return errors.New(errMsg)
	}

	fmt.Print(out.String())

	return nil
}
