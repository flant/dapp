package stage

import "github.com/flant/werf/pkg/container_runtime"

func newGitStage(name StageName, baseStageOptions *NewBaseStageOptions) *GitStage {
	s := &GitStage{}
	s.BaseStage = newBaseStage(name, baseStageOptions)
	return s
}

type GitStage struct {
	*BaseStage
}

func (s *GitStage) IsEmpty(_ Conveyor, _ container_runtime.ImageInterface) (bool, error) {
	return s.isEmpty(), nil
}

func (s *GitStage) isEmpty() bool {
	return len(s.gitMappings) == 0
}

func (s *GitStage) AfterImageSyncDockerStateHook(c Conveyor) error {
	if s.image.GetImageInfo() == nil {
		stageName := c.GetBuildingGitStage(s.imageName)
		if stageName == "" {
			c.SetBuildingGitStage(s.imageName, s.Name())
		}
	}

	return nil
}

func (s *GitStage) PrepareImage(c Conveyor, prevBuiltImage, image container_runtime.ImageInterface) error {
	if err := s.BaseStage.PrepareImage(c, prevBuiltImage, image); err != nil {
		return err
	}

	return nil
}
