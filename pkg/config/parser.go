package config

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"

	"github.com/flant/werf/pkg/git_repo"
	"github.com/flant/werf/pkg/logger"
	"github.com/flant/werf/pkg/slug"
	"github.com/flant/werf/pkg/util"
	"gopkg.in/flant/yaml.v2"
)

func ParseWerfConfig(werfConfigPath string) (*WerfConfig, error) {
	werfConfigRenderContent, err := parseWerfConfigYaml(werfConfigPath)
	if err != nil {
		return nil, err
	}

	werfConfigRenderPath, err := dumpWerfConfigRender(werfConfigPath, werfConfigRenderContent)
	if err != nil {
		return nil, err
	}

	docs, err := splitByDocs(werfConfigRenderContent, werfConfigRenderPath)
	if err != nil {
		return nil, err
	}

	meta, rawImages, err := splitByMetaAndRawImages(docs)
	if err != nil {
		return nil, err
	}

	if meta == nil {
		defaultProjectName, err := GetProjectName(path.Dir(werfConfigPath))
		if err != nil {
			return nil, err
		}

		format := "meta definition is not defined: add meta doc with required fields, e.g:\n\n" +
			"```\n" +
			"project: %s\n" +
			"---\n" +
			"```\n\n" +
			"##################################################################################################################\n" +
			"###     WARNING! Project name cannot be changed later without rebuilding and redeploying your application!     ###\n" +
			"###  Read more about meta doc here, https://flant.github.io/werf/reference/config.html#meta-configuration-doc  ###\n" +
			"##################################################################################################################"

		return nil, fmt.Errorf(format, defaultProjectName)
	}

	images, err := splitByImages(rawImages, werfConfigRenderContent, werfConfigRenderPath)
	if err != nil {
		return nil, err
	}

	werfConfig := &WerfConfig{
		Meta:   meta,
		Images: images,
	}

	return werfConfig, nil
}

func GetProjectName(projectDir string) (string, error) {
	name := path.Base(projectDir)

	if exist, err := util.DirExists(path.Join(projectDir, ".git")); err != nil {
		return "", err
	} else if exist {
		remoteOriginUrl, err := gitOwnRepoOriginUrl(projectDir)
		if err != nil {
			return "", err
		}

		if remoteOriginUrl != "" {
			ep, err := transport.NewEndpoint(remoteOriginUrl)
			if err != nil {
				return "", fmt.Errorf("bad url '%s': %s", remoteOriginUrl, err)
			}

			gitName := strings.TrimSuffix(ep.Path, ".git")

			return slug.Project(gitName), nil
		}
	}

	return slug.Project(name), nil
}

func gitOwnRepoOriginUrl(projectDir string) (string, error) {
	localGitRepo := &git_repo.Local{
		Path:   projectDir,
		GitDir: path.Join(projectDir, ".git"),
	}

	remoteOriginUrl, err := localGitRepo.RemoteOriginUrl()
	if err != nil {
		return "", nil
	}

	return remoteOriginUrl, nil
}

func dumpWerfConfigRender(werfConfigPath string, werfConfigRenderContent string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	werfConfigNameParts := strings.Split(path.Base(werfConfigPath), ".")
	var werfConfigRenderNameParts []string
	werfConfigRenderNameParts = append(werfConfigRenderNameParts, werfConfigNameParts[0:len(werfConfigNameParts)-1]...)
	werfConfigRenderNameParts = append(werfConfigRenderNameParts, "render", werfConfigNameParts[len(werfConfigNameParts)-1])
	werfConfigRenderPath := path.Join(wd, fmt.Sprintf(".%s", strings.Join(werfConfigRenderNameParts, ".")))

	werfConfigRenderFile, err := os.OpenFile(werfConfigRenderPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return "", err
	}
	werfConfigRenderFile.Write([]byte(werfConfigRenderContent))
	werfConfigRenderFile.Close()

	return werfConfigRenderPath, nil
}

func splitByDocs(werfConfigRenderContent string, werfConfigRenderPath string) ([]*doc, error) {
	var docs []*doc
	var line int
	for _, docContent := range splitContent([]byte(werfConfigRenderContent)) {
		if !emptyDocContent(docContent) {
			docs = append(docs, &doc{
				Line:           line,
				Content:        docContent,
				RenderFilePath: werfConfigRenderPath,
			})
		}

		contentLines := bytes.Split(docContent, []byte("\n"))
		if string(contentLines[len(contentLines)-1]) == "" {
			contentLines = contentLines[0 : len(contentLines)-1]
		}
		line += len(contentLines) + 1
	}

	return docs, nil
}

func parseWerfConfigYaml(werfConfigPath string) (string, error) {
	data, err := ioutil.ReadFile(werfConfigPath)
	if err != nil {
		return "", err
	}

	tmpl := template.New("werfConfig")
	tmpl.Funcs(funcMap(tmpl))

	projectDir := filepath.Dir(werfConfigPath)
	werfConfigsDir := filepath.Join(projectDir, ".werf")
	werfConfigsTemplates, err := getWerfConfigsTemplates(werfConfigsDir)
	if err != nil {
		return "", err
	}

	if len(werfConfigsTemplates) != 0 {
		for _, templatePath := range werfConfigsTemplates {
			templateName, err := filepath.Rel(werfConfigsDir, templatePath)
			if err != nil {
				return "", err
			}

			extraTemplate := tmpl.New(templateName)

			var filePathData []byte
			if filePathData, err = ioutil.ReadFile(templatePath); err != nil {
				return "", err
			}

			if _, err := extraTemplate.Parse(string(filePathData)); err != nil {
				return "", err
			}
		}
	}

	if _, err := tmpl.Parse(string(data)); err != nil {
		return "", err
	}

	files := files{filepath.Dir(werfConfigPath)}
	config, err := executeTemplate(tmpl, "werfConfig", map[string]interface{}{"Files": files})

	return config, err
}

func getWerfConfigsTemplates(path string) ([]string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil
	}

	var templates []string
	err := filepath.Walk(path, func(fp string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if fi.IsDir() {
			return nil
		}

		matched, err := filepath.Match("*.tmpl", fi.Name())
		if err != nil {
			return err
		}

		if matched {
			templates = append(templates, fp)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return templates, nil
}

func funcMap(tmpl *template.Template) template.FuncMap {
	funcMap := sprig.TxtFuncMap()
	funcMap["include"] = func(name string, data interface{}) (string, error) {
		return executeTemplate(tmpl, name, data)
	}
	return funcMap
}

func executeTemplate(tmpl *template.Template, name string, data interface{}) (string, error) {
	buf := bytes.NewBuffer(nil)
	if err := tmpl.ExecuteTemplate(buf, name, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

type files struct {
	HomePath string
}

func (f files) Get(path string) string {
	filePath := filepath.Join(f.HomePath, path)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		logger.LogErrorF("WARNING: Config: {{ .Files.Get '%s' }}: file '%s' not exist!\n", path, filePath)
		return ""
	}

	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		return ""
	}
	return string(b)
}

func splitContent(content []byte) (docsContents [][]byte) {
	const (
		stateLineBegin   = "stateLineBegin"
		stateRegularLine = "stateRegularLine"
		stateDocDash1    = "stateDocDash1"
		stateDocDash2    = "stateDocDash2"
		stateDocDash3    = "stateDocDash3"
		stateDocSpaces   = "stateDocSpaces"
		stateDocComment  = "stateDocComment"
	)

	state := stateLineBegin
	var docStartIndex, separatorLength int
	var docContent []byte
	var index int
	var ch byte
	for index, ch = range content {
		switch ch {
		case '-':
			switch state {
			case stateLineBegin:
				separatorLength = 1
				state = stateDocDash1
			case stateDocDash1, stateDocDash2:
				separatorLength += 1

				switch state {
				case stateDocDash1:
					state = stateDocDash2
				case stateDocDash2:
					state = stateDocDash3
				}
			default:
				state = stateRegularLine
			}
		case '\n':
			switch state {
			case stateDocDash3, stateDocSpaces, stateDocComment:
				if docStartIndex == index-separatorLength {
					docContent = []byte{}
				} else {
					docContent = content[docStartIndex : index-separatorLength]
				}
				docsContents = append(docsContents, docContent)
				docStartIndex = index + 1
			}
			separatorLength = 0
			state = stateLineBegin
		case ' ', '\r', '\t':
			switch state {
			case stateDocDash3, stateDocSpaces:
				separatorLength += 1
				state = stateDocSpaces
			case stateDocComment:
				separatorLength += 1
			default:
				state = stateRegularLine
			}
		case '#':
			switch state {
			case stateDocDash3, stateDocSpaces, stateDocComment:
				separatorLength += 1
				state = stateDocComment
			default:
				state = stateRegularLine
			}
		default:
			switch state {
			case stateDocComment:
				separatorLength += 1
			default:
				state = stateRegularLine
			}
		}
	}

	if docStartIndex != index+1 {
		switch state {
		case stateDocDash3, stateDocSpaces, stateDocComment:
			separatorLengthWithoutCursor := separatorLength - 1
			if docStartIndex == index-separatorLengthWithoutCursor {
				docContent = []byte{}
			} else {
				docContent = content[docStartIndex : index-separatorLengthWithoutCursor]
			}
		default:
			docContent = content[docStartIndex:]
		}
		docsContents = append(docsContents, docContent)
	}

	return docsContents
}

func emptyDocContent(content []byte) bool {
	const (
		stateRegular = 0
		stateComment = 1
	)

	state := stateRegular
	for _, ch := range content {
		switch ch {
		case '#':
			state = stateComment
		case '\n':
			state = stateRegular
		case ' ', '\r', '\t':
		default:
			if state == stateRegular {
				return false
			}
		}
	}
	return true
}

func splitByImages(rawImages []*rawImage, werfConfigRenderContent string, werfConfigRenderPath string) ([]*Image, error) {
	var images []*Image
	var artifacts []*ImageArtifact

	for _, rawImage := range rawImages {
		if rawImage.imageType() == "images" {
			if sameImages, err := rawImage.toImageDirectives(); err != nil {
				return nil, err
			} else {
				images = append(images, sameImages...)
			}
		} else {
			if imageArtifact, err := rawImage.toImageArtifactDirective(); err != nil {
				return nil, err
			} else {
				artifacts = append(artifacts, imageArtifact)
			}
		}
	}

	if len(images) == 0 {
		return nil, newConfigError(fmt.Sprintf("no images defined, at least one image required!\n\n%s:\n\n```\n%s```\n", werfConfigRenderPath, werfConfigRenderContent))
	}

	if err := exportsAutoExcluding(images, artifacts); err != nil {
		return nil, err
	}

	if err := validateImagesNames(images, artifacts); err != nil {
		return nil, err
	}

	if err := associateImportsArtifacts(images, artifacts); err != nil {
		return nil, err
	}

	if err := associateImagesFrom(images, artifacts); err != nil {
		return nil, err
	}

	return images, nil
}

func exportsAutoExcluding(images []*Image, artifacts []*ImageArtifact) error {
	for _, image := range images {
		if err := image.exportsAutoExcluding(); err != nil {
			return err
		}
	}

	for _, artifact := range artifacts {
		if err := artifact.exportsAutoExcluding(); err != nil {
			return err
		}
	}

	return nil
}

func validateImagesNames(images []*Image, artifacts []*ImageArtifact) error {
	existByImageName := map[string]bool{}

	imageByName := map[string]*Image{}
	for _, image := range images {
		name := image.Name

		if d, ok := imageByName[name]; ok {
			return newConfigError(fmt.Sprintf("conflict between images names!\n\n%s%s\n", dumpConfigDoc(d.raw.doc), dumpConfigDoc(image.raw.doc)))
		} else {
			imageByName[name] = image
			existByImageName[name] = true
		}
	}

	imageArtifactByName := map[string]*ImageArtifact{}
	for _, artifact := range artifacts {
		name := artifact.Name

		if a, ok := imageArtifactByName[name]; ok {
			return newConfigError(fmt.Sprintf("conflict between artifacts names!\n\n%s%s\n", dumpConfigDoc(a.raw.doc), dumpConfigDoc(artifact.raw.doc)))
		} else {
			imageArtifactByName[name] = artifact
		}

		if exist, ok := existByImageName[name]; ok && exist {
			d := imageByName[name]

			return newConfigError(fmt.Sprintf("conflict between image and artifact names!\n\n%s%s\n", dumpConfigDoc(d.raw.doc), dumpConfigDoc(artifact.raw.doc)))
		} else {
			imageArtifactByName[name] = artifact
		}
	}

	return nil
}

func associateImportsArtifacts(images []*Image, artifacts []*ImageArtifact) error {
	var artifactImports []*ArtifactImport

	for _, image := range images {
		for _, relatedImageInterface := range image.relatedImages() {
			switch relatedImageInterface.(type) {
			case *Image:
				artifactImports = append(artifactImports, relatedImageInterface.(*Image).Import...)
			case *ImageArtifact:
				artifactImports = append(artifactImports, relatedImageInterface.(*ImageArtifact).Import...)
			}
		}
	}

	for _, artifactImage := range artifacts {
		for _, relatedImageInterface := range artifactImage.relatedImages() {
			switch relatedImageInterface.(type) {
			case *Image:
				artifactImports = append(artifactImports, relatedImageInterface.(*Image).Import...)
			case *ImageArtifact:
				artifactImports = append(artifactImports, relatedImageInterface.(*ImageArtifact).Import...)
			}
		}
	}

	for _, artifactImport := range artifactImports {
		if err := artifactImport.associateArtifact(artifacts); err != nil {
			return err
		}
	}

	return nil
}

func associateImagesFrom(images []*Image, artifacts []*ImageArtifact) error {
	for _, image := range images {
		if err := associateImageFrom(image.lastLayerOrSelf(), images, artifacts); err != nil {
			return err
		}
	}

	for _, image := range artifacts {
		if err := associateImageFrom(image.lastLayerOrSelf(), images, artifacts); err != nil {
			return err
		}
	}

	return nil
}

func associateImageFrom(image ImageInterface, images []*Image, artifacts []*ImageArtifact) error {
	switch image.(type) {
	case *Image:
		return image.(*Image).associateFrom(images, artifacts)
	case *ImageArtifact:
		return image.(*ImageArtifact).associateFrom(images, artifacts)
	default:
		panic("runtime error")
	}
}

func splitByMetaAndRawImages(docs []*doc) (*Meta, []*rawImage, error) {
	var rawImages []*rawImage
	var resultMeta *Meta

	parentStack = util.NewStack()
	for _, doc := range docs {
		var raw map[string]interface{}
		err := yaml.Unmarshal(doc.Content, &raw)
		if err != nil {
			return nil, nil, newYamlUnmarshalError(err, doc)
		}

		if isMetaDoc(raw) {
			if resultMeta != nil {
				return nil, nil, newYamlUnmarshalError(errors.New("duplicate meta definition"), doc)
			}

			rawMeta := &rawMeta{doc: doc}
			err := yaml.Unmarshal(doc.Content, &rawMeta)
			if err != nil {
				return nil, nil, newYamlUnmarshalError(err, doc)
			}

			resultMeta = rawMeta.toMeta()
		} else if isImageDoc(raw) {
			image := &rawImage{doc: doc}
			err := yaml.Unmarshal(doc.Content, &image)
			if err != nil {
				return nil, nil, newYamlUnmarshalError(err, doc)
			}

			rawImages = append(rawImages, image)
		} else {
			return nil, nil, newYamlUnmarshalError(errors.New("doc type cannot be recognized"), doc)
		}
	}

	return resultMeta, rawImages, nil
}

func isMetaDoc(h map[string]interface{}) bool {
	if _, ok := h["project"]; ok {
		return true
	}

	return false
}

func isImageDoc(h map[string]interface{}) bool {
	if _, ok := h["image"]; ok {
		return true
	} else if _, ok := h["artifact"]; ok {
		return true
	}

	return false
}

func newYamlUnmarshalError(err error, doc *doc) error {
	switch err.(type) {
	case *configError:
		return err
	default:
		message := err.Error()
		reg, err := regexp.Compile("line ([0-9]+)")
		if err != nil {
			return err
		}

		res := reg.FindStringSubmatch(message)

		if len(res) == 2 {
			line, err := strconv.Atoi(res[1])
			if err != nil {
				return err
			}

			message = reg.ReplaceAllString(message, fmt.Sprintf("line %d", line+doc.Line))
		}
		return newDetailedConfigError(message, nil, doc)
	}
}
