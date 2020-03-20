package stage

import (
	"fmt"

	"github.com/flant/werf/pkg/build/builder"
	"github.com/flant/werf/pkg/container_runtime"
	"github.com/flant/werf/pkg/image"
)

func newUserWithGitPatchStage(builder builder.Builder, name StageName, gitPatchStageOptions *NewGitPatchStageOptions, baseStageOptions *NewBaseStageOptions) *UserWithGitPatchStage {
	s := &UserWithGitPatchStage{}
	s.UserStage = newUserStage(builder, name, baseStageOptions)
	s.GitPatchStage = newGitPatchStage(name, gitPatchStageOptions, baseStageOptions)
	s.GitPatchStage.BaseStage = s.BaseStage

	return s
}

type UserWithGitPatchStage struct {
	*UserStage
	GitPatchStage *GitPatchStage
}

func (s *UserWithGitPatchStage) SelectCacheImage(images []*image.Info) (*image.Info, error) {
	ancestorsImages, err := s.selectCacheImagesAncestorsByGitMappings(images)
	if err != nil {
		return nil, fmt.Errorf("unable to select cache images ancestors by git mappings: %s", err)
	}
	return s.selectCacheImageByOldestCreationTimestamp(ancestorsImages)
}

func (s *UserWithGitPatchStage) GetNextStageDependencies(c Conveyor) (string, error) {
	return s.BaseStage.getNextStageGitDependencies(c)
}

func (s *UserWithGitPatchStage) PrepareImage(c Conveyor, prevBuiltImage, image container_runtime.ImageInterface) error {
	if err := s.BaseStage.PrepareImage(c, prevBuiltImage, image); err != nil {
		return err
	}

	if !s.GitPatchStage.isEmpty() {
		stageName := c.GetBuildingGitStage(s.imageName)
		if stageName == s.Name() {
			if err := s.GitPatchStage.prepareImage(c, prevBuiltImage, image); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *UserWithGitPatchStage) AfterImageSyncDockerStateHook(c Conveyor) error {
	if !s.GitPatchStage.isEmpty() {
		if err := s.GitPatchStage.AfterImageSyncDockerStateHook(c); err != nil {
			return err
		}
	}

	return nil
}
