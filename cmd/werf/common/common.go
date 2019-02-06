package common

import (
	"errors"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/kubernetes/pkg/util/file"

	"github.com/flant/kubedog/pkg/kube"
	"github.com/flant/werf/pkg/config"
	"github.com/flant/werf/pkg/logger"
	"github.com/flant/werf/pkg/werf"
)

type CmdData struct {
	Dir     *string
	TmpDir  *string
	HomeDir *string
	SSHKeys *[]string

	Tag          *[]string
	TagGitBranch *string
	TagGitTag    *string
	TagGitCommit *string

	Environment *string
	Release     *string
	Namespace   *string
	KubeContext *string

	StagesRepo     *string
	StagesUsername *string
	StagesPassword *string

	ImagesRepo     *string
	ImagesUsername *string
	ImagesPassword *string

	PullUsername *string
	PullPassword *string

	DryRun bool
}

func GetLongCommandDescription(text string) string {
	return logger.FitTextWithIndentWithWidthMaxLimit(text, 0, 100)
}

func SetupDir(cmdData *CmdData, cmd *cobra.Command) {
	cmdData.Dir = new(string)
	cmd.Flags().StringVarP(cmdData.Dir, "dir", "", "", "Change to the specified directory to find werf.yaml config")
}

func SetupTmpDir(cmdData *CmdData, cmd *cobra.Command) {
	cmdData.TmpDir = new(string)
	cmd.Flags().StringVarP(cmdData.TmpDir, "tmp-dir", "", "", "Use specified dir to store tmp files and dirs (use WERF_TMP environment or system tmp dir by default)")
}

func SetupHomeDir(cmdData *CmdData, cmd *cobra.Command) {
	cmdData.HomeDir = new(string)
	cmd.Flags().StringVarP(cmdData.HomeDir, "home-dir", "", "", "Use specified dir to store werf cache files and dirs (use WERF_HOME environment or ~/.werf by default)")
}

func SetupSSHKey(cmdData *CmdData, cmd *cobra.Command) {
	cmdData.SSHKeys = new([]string)
	cmd.Flags().StringArrayVarP(cmdData.SSHKeys, "ssh-key", "", []string{}, "Enable only specified ssh keys (use system ssh-agent by default)")
}

func SetupTag(cmdData *CmdData, cmd *cobra.Command) {
	cmdData.Tag = new([]string)
	cmdData.TagGitBranch = new(string)
	cmdData.TagGitTag = new(string)
	cmdData.TagGitCommit = new(string)

	cmd.Flags().StringArrayVarP(cmdData.Tag, "tag", "", []string{}, "Add tag (can be used one or more times)")
	cmd.Flags().StringVarP(cmdData.TagGitBranch, "tag-git-branch", "", os.Getenv("WERF_AUTOTAG_GIT_BRANCH"), "Tag by git branch (use WERF_AUTOTAG_GIT_BRANCH environment by default)")
	cmd.Flags().StringVarP(cmdData.TagGitTag, "tag-git-tag", "", os.Getenv("WERF_AUTOTAG_GIT_TAG"), "Tag by git tag (use WERF_AUTOTAG_GIT_TAG environment by default)")
	cmd.Flags().StringVarP(cmdData.TagGitCommit, "tag-git-commit", "", os.Getenv("WERF_AUTOTAG_GIT_COMMIT"), "Tag by git commit (use WERF_AUTOTAG_GIT_COMMIT environment by default)")
}

func SetupEnvironment(cmdData *CmdData, cmd *cobra.Command) {
	cmdData.Environment = new(string)
	cmd.Flags().StringVarP(cmdData.Environment, "env", "", "", "Use specified environment (use WERF_DEPLOY_ENVIRONMENT by default)")
}

func SetupRelease(cmdData *CmdData, cmd *cobra.Command) {
	cmdData.Release = new(string)
	cmd.Flags().StringVarP(cmdData.Release, "release", "", "", "Use specified Helm release name (use %project-%environment template by default)")
}

func SetupNamespace(cmdData *CmdData, cmd *cobra.Command) {
	cmdData.Namespace = new(string)
	cmd.Flags().StringVarP(cmdData.Namespace, "namespace", "", "", "Use specified Kubernetes namespace (use %project-%environment template by default)")
}

func SetupKubeContext(cmdData *CmdData, cmd *cobra.Command) {
	cmdData.KubeContext = new(string)
	cmd.Flags().StringVarP(cmdData.KubeContext, "kube-context", "", "", "Kubernetes config context")
}

func SetupStagesRepo(cmdData *CmdData, cmd *cobra.Command) {
	cmdData.StagesRepo = new(string)
	cmd.Flags().StringVarP(cmdData.StagesRepo, "stages", "s", "", "Docker Repo to store stages or :local for non-distributed build (only :local is supported for now)")
}

func SetupStagesUsername(cmdData *CmdData, cmd *cobra.Command) {
	cmdData.StagesUsername = new(string)
	cmd.Flags().StringVarP(cmdData.StagesUsername, "stages-username", "", "", "Stages Docker repo username")
}

func SetupStagesPassword(cmdData *CmdData, cmd *cobra.Command) {
	cmdData.StagesPassword = new(string)
	cmd.Flags().StringVarP(cmdData.StagesPassword, "stages-password", "", "", "Stages Docker repo password")
}

func SetupPullUsername(cmdData *CmdData, cmd *cobra.Command) {
	cmdData.PullUsername = new(string)
	cmd.Flags().StringVarP(cmdData.PullUsername, "pull-username", "", "", "Username to authorize pull of base images")
}

func SetupPullPassword(cmdData *CmdData, cmd *cobra.Command) {
	cmdData.PullPassword = new(string)
	cmd.Flags().StringVarP(cmdData.PullPassword, "pull-password", "", "", "Password to authorize pull of base images")
}

func SetupImagesRepo(cmdData *CmdData, cmd *cobra.Command) {
	cmdData.ImagesRepo = new(string)
	cmd.Flags().StringVarP(cmdData.ImagesRepo, "images", "i", os.Getenv("WERF_IMAGES_REPO"), "Docker Repo to store images (use WERF_IMAGES_REPO environment by default)")
}

func SetupCleanupImagesUsername(cmdData *CmdData, cmd *cobra.Command) {
	usage := `Images Docker repo username (granted permission to read images info and delete images).
Use by default:
* werf-cleanup username if WERF_IMAGES_PASSWORD environment defined or 
* WERF_IMAGES_USERNAME environment`

	if os.Getenv("WERF_CLEANUP_IMAGES_PASSWORD") != "" {
		setupImagesUsername(cmdData, cmd, "werf-cleanup", usage)
		return
	}

	setupImagesUsername(cmdData, cmd, os.Getenv("WERF_IMAGES_USERNAME"), usage)
}

func SetupCleanupImagesPassword(cmdData *CmdData, cmd *cobra.Command) {
	usage := `Docker repo password (granted permission to read images info and delete images).
Use by default:
* WERF_CLEANUP_IMAGES_PASSWORD or 
* WERF_IMAGES_PASSWORD environment`

	if os.Getenv("WERF_CLEANUP_IMAGES_PASSWORD") != "" {
		setupImagesPassword(cmdData, cmd, os.Getenv("WERF_CLEANUP_IMAGES_PASSWORD"), usage)
		return
	}

	setupImagesPassword(cmdData, cmd, os.Getenv("WERF_IMAGES_PASSWORD"), usage)
}

func SetupImagesUsernameWithUsage(cmdData *CmdData, cmd *cobra.Command, usage string) {
	setupImagesUsername(cmdData, cmd, os.Getenv("WERF_IMAGES_USERNAME"), usage)
}

func SetupImagesPasswordWithUsage(cmdData *CmdData, cmd *cobra.Command, usage string) {
	setupImagesPassword(cmdData, cmd, os.Getenv("WERF_IMAGES_PASSWORD"), usage)
}

func setupImagesUsername(cmdData *CmdData, cmd *cobra.Command, value, usage string) {
	cmdData.ImagesUsername = new(string)
	cmd.Flags().StringVarP(cmdData.ImagesUsername, "images-username", "u", value, usage)
}

func setupImagesPassword(cmdData *CmdData, cmd *cobra.Command, value, usage string) {
	cmdData.ImagesPassword = new(string)
	cmd.Flags().StringVarP(cmdData.ImagesPassword, "images-password", "p", value, usage)
}

func SetupDryRun(cmdData *CmdData, cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&cmdData.DryRun, "dry-run", "", false, "Indicate what the command would do without actually doing that")
}

func GetStagesRepo(cmdData *CmdData) (string, error) {
	if *cmdData.StagesRepo == "" {
		return "", fmt.Errorf("--stages :local param required")
	} else if *cmdData.StagesRepo != ":local" {
		return "", fmt.Errorf("only --stages :local is supported for now, got '%s'", *cmdData.StagesRepo)
	}
	return *cmdData.StagesRepo, nil
}

func GetImagesRepo(projectName string, cmdData *CmdData) (string, error) {
	if *cmdData.ImagesRepo == "" {
		return "", fmt.Errorf("--images REPO param required")
	}
	return GetOptionalImagesRepo(projectName, cmdData), nil
}

func GetOptionalImagesRepo(projectName string, cmdData *CmdData) string {
	repoOption := *cmdData.ImagesRepo

	if repoOption == ":minikube" {
		return fmt.Sprintf("werf-registry.kube-system.svc.cluster.local:5000/%s", projectName)
	} else if repoOption != "" {
		return repoOption
	}

	return ""
}

func GetWerfConfig(projectDir string) (*config.WerfConfig, error) {
	for _, werfConfigName := range []string{"werf.yml", "werf.yaml"} {
		werfConfigPath := path.Join(projectDir, werfConfigName)
		if exist, err := file.FileExists(werfConfigPath); err != nil {
			return nil, err
		} else if exist {
			return config.ParseWerfConfig(werfConfigPath)
		}
	}

	return nil, errors.New("werf.yaml not found")
}

func GetProjectDir(cmdData *CmdData) (string, error) {
	if *cmdData.Dir != "" {
		return *cmdData.Dir, nil
	}

	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	return currentDir, nil
}

func GetNamespace(namespaceOption string) string {
	if namespaceOption == "" {
		return kube.DefaultNamespace
	}
	return namespaceOption
}

func GetKubeContext(kubeContextOption string) string {
	kubeContext := os.Getenv("KUBECONTEXT")
	if kubeContext == "" {
		return kubeContextOption
	}
	return kubeContext
}

func LogRunningTime(f func() error) error {
	t := time.Now()
	err := f()

	logger.LogServiceLn(fmt.Sprintf("Running time %0.2f seconds", time.Now().Sub(t).Seconds()))

	return err
}

func LogVersion() {
	logger.LogInfoF("Version: %s\n", werf.Version)
}

func LogProjectDir(dir string) {
	if os.Getenv("WERF_LOG_PROJECT_DIR") != "" {
		logger.LogInfoF("Using project dir: %s\n", dir)
	}
}
