package helm

import (
	"bytes"
	"fmt"
	"gopkg.in/yaml.v2"
	"strings"

	"k8s.io/helm/pkg/releaseutil"

	"github.com/flant/werf/pkg/util"
)

type ChartTemplates []Template

func (templates ChartTemplates) Pods() []Template {
	return templates.ByKind("Pod")
}

func (templates ChartTemplates) Jobs() []Template {
	return templates.ByKind("Job")
}

func (templates ChartTemplates) Deployments() []Template {
	return templates.ByKind("Deployment")
}

func (templates ChartTemplates) StatefulSets() []Template {
	return templates.ByKind("StatefulSet")
}

func (templates ChartTemplates) DaemonSets() []Template {
	return templates.ByKind("DaemonSet")
}

func (templates ChartTemplates) ByKind(kind string) []Template {
	var resultTemplates []Template

	for _, template := range templates {
		if strings.ToLower(template.Kind) == strings.ToLower(kind) {
			resultTemplates = append(resultTemplates, template)
		}
	}

	return resultTemplates
}

type Template struct {
	Version  string `yaml:"apiVersion"`
	Kind     string `yaml:"kind,omitempty"`
	Metadata struct {
		Name        string            `yaml:"name"`
		Namespace   string            `yaml:"namespace"`
		Annotations map[string]string `yaml:"annotations"`
		UID         string            `yaml:"uid"`
	} `yaml:"metadata,omitempty"`
	Status string `yaml:"status,omitempty"`
}

func (t Template) Namespace(namespace string) string {
	if t.Metadata.Namespace != "" {
		return t.Metadata.Namespace
	}

	return namespace
}

func GetTemplatesFromRevision(releaseName string, revision int32) (ChartTemplates, error) {
	rawTemplates, err := getRawTemplatesFromRevision(releaseName, revision)
	if err != nil {
		return nil, err
	}

	chartTemplates, err := parseTemplates(rawTemplates)
	if err != nil {
		return nil, fmt.Errorf("unable to parse revision templates: %s", err)
	}

	return chartTemplates, nil
}

func GetTemplatesFromChart(chartPath, releaseName, namespace string, values, set, setString []string) (ChartTemplates, error) {
	rawTemplates, err := getRawTemplatesFromChart(chartPath, releaseName, namespace, values, set, setString)
	if err != nil {
		return nil, err
	}

	chartTemplates, err := parseTemplates(rawTemplates)
	if err != nil {
		return nil, fmt.Errorf("unable to parse chart templates: %s", err)
	}

	return chartTemplates, nil
}

func getRawTemplatesFromChart(chartPath, releaseName, namespace string, values, set, setString []string) (string, error) {
	out := &bytes.Buffer{}

	renderOptions := RenderOptions{
		ShowNotes: false,
	}

	if err := Render(out, chartPath, releaseName, namespace, values, set, setString, renderOptions); err != nil {
		return "", err
	}

	return out.String(), nil
}

func getRawTemplatesFromRevision(releaseName string, revision int32) (string, error) {
	var result string
	resp, err := releaseContent(releaseName, releaseContentOptions{Version: revision})
	if err != nil {
		return "", err
	}

	for _, hook := range resp.Release.Hooks {
		result += fmt.Sprintf("---\n# %s\n%s\n", hook.Name, hook.Manifest)
	}

	result += "\n"
	result += resp.Release.Manifest

	return result, nil
}

func parseTemplates(rawTemplates string) (ChartTemplates, error) {
	var templates ChartTemplates

	for _, doc := range releaseutil.SplitManifests(rawTemplates) {
		var t Template
		err := yaml.Unmarshal([]byte(doc), &t)
		if err != nil {
			return nil, fmt.Errorf("%s\n\n%s\n", err, util.NumerateLines(doc, 1))
		}

		if t.Metadata.Name != "" {
			templates = append(templates, t)
		}
	}

	return templates, nil
}
