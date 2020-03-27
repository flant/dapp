package build

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/flant/werf/pkg/container_runtime"
	"github.com/flant/werf/pkg/util"

	"github.com/docker/docker/pkg/stringid"

	"github.com/flant/logboek"

	"github.com/flant/werf/pkg/build/stage"
	"github.com/flant/werf/pkg/image"
	imagePkg "github.com/flant/werf/pkg/image"
	"github.com/flant/werf/pkg/stapel"
	"github.com/flant/werf/pkg/werf"
)

type BuildPhaseOptions struct {
	CalculateStagesOnly bool
	ImageBuildOptions   container_runtime.BuildOptions
	IntrospectOptions   IntrospectOptions
}

type BuildStagesOptions struct {
	ImageBuildOptions container_runtime.BuildOptions
	IntrospectOptions
}

type IntrospectOptions struct {
	Targets []IntrospectTarget
}

type IntrospectTarget struct {
	ImageName string
	StageName string
}

func (opts *IntrospectOptions) ImageStageShouldBeIntrospected(imageName, stageName string) bool {
	for _, s := range opts.Targets {
		if (s.ImageName == "*" || s.ImageName == imageName) && s.StageName == stageName {
			return true
		}
	}

	return false
}

func NewBuildPhase(c *Conveyor, opts BuildPhaseOptions) *BuildPhase {
	return &BuildPhase{
		BasePhase:         BasePhase{c},
		BuildPhaseOptions: opts,
	}
}

type BuildPhase struct {
	BasePhase
	BuildPhaseOptions

	StagesIterator              *StagesIterator
	ShouldAddManagedImageRecord bool
}

func (phase *BuildPhase) Name() string {
	return "build"
}

func (phase *BuildPhase) BeforeImages() error {
	return nil
}

func (phase *BuildPhase) AfterImages() error {
	return nil
}

func (phase *BuildPhase) ImageProcessingShouldBeStopped(img *Image) bool {
	return false
}

func (phase *BuildPhase) BeforeImageStages(img *Image) error {
	phase.StagesIterator = NewStagesIterator(phase.Conveyor)

	img.SetupBaseImage(phase.Conveyor)

	return nil
}

func (phase *BuildPhase) AfterImageStages(img *Image) error {
	img.SetLastNonEmptyStage(phase.StagesIterator.PrevNonEmptyStage)

	stagesSig, err := calculateSignature("imageStages", "", phase.StagesIterator.PrevNonEmptyStage, phase.Conveyor)
	if err != nil {
		return fmt.Errorf("unable to calculate image %s stages-signature: %s", img.GetName(), err)
	}
	img.SetStagesSignature(stagesSig)

	if phase.ShouldAddManagedImageRecord {
		if err := phase.Conveyor.StagesStorage.AddManagedImage(phase.Conveyor.projectName(), img.GetName()); err != nil {
			return fmt.Errorf("unable to add image %q to the managed images of project %q: %s", img.GetName(), phase.Conveyor.projectName(), err)
		}
	}

	return nil
}

func (phase *BuildPhase) getPrevNonEmptyStageImageSize() int64 {
	if phase.StagesIterator.PrevNonEmptyStage != nil {
		if phase.StagesIterator.PrevNonEmptyStage.GetImage().GetStagesStorageImageInfo() != nil {
			return phase.StagesIterator.PrevNonEmptyStage.GetImage().GetStagesStorageImageInfo().Size
		}
	}
	return 0
}

func (phase *BuildPhase) OnImageStage(img *Image, stg stage.Interface) error {
	return phase.StagesIterator.OnImageStage(img, stg, func(img *Image, stg stage.Interface, isEmpty bool) error {
		return phase.onImageStage(img, stg, isEmpty)
	})
}

func (phase *BuildPhase) onImageStage(img *Image, stg stage.Interface, isEmpty bool) error {
	if isEmpty {
		return nil
	}

	if phase.CalculateStagesOnly {
		return phase.calculateStage(img, stg)
	} else {
		if stg.Name() != "from" {
			if phase.StagesIterator.PrevNonEmptyStage == nil {
				panic(fmt.Sprintf("expected PrevNonEmptyStage to be set for image %q stage %s", img.GetName(), stg.Name()))
			}
			if phase.StagesIterator.PrevBuiltStage == nil {
				panic(fmt.Sprintf("expected PrevBuiltStage to be set for image %q stage %s", img.GetName(), stg.Name()))
			}
			if phase.StagesIterator.PrevBuiltStage != phase.StagesIterator.PrevNonEmptyStage {
				panic(fmt.Sprintf("expected PrevBuiltStage (%q) to equal PrevNonEmptyStage (%q) for image %q stage %s", phase.StagesIterator.PrevBuiltStage.LogDetailedName(), phase.StagesIterator.PrevNonEmptyStage.LogDetailedName(), img.GetName(), stg.Name()))
			}
		}

		if err := phase.calculateStage(img, stg); err != nil {
			return err
		}

		// Stage is cached in the stages storage
		if stg.GetImage().GetStagesStorageImageInfo() != nil {
			logboek.Default.LogFHighlight("Use cache image for %s\n", stg.LogDetailedName())

			logImageInfo(stg.GetImage(), phase.getPrevNonEmptyStageImageSize(), true)

			logboek.LogOptionalLn()

			if phase.IntrospectOptions.ImageStageShouldBeIntrospected(img.GetName(), string(stg.Name())) {
				if err := introspectStage(stg); err != nil {
					return err
				}
			}

			return nil
		}

		if err := phase.fetchBaseImageForStage(img, stg); err != nil {
			return err
		}

		if err := phase.prepareStageInstructions(img, stg); err != nil {
			return err
		}

		if err := phase.buildStage(img, stg); err != nil {
			return err
		}

		if stg.GetImage().GetStagesStorageImageInfo() == nil {
			panic(fmt.Sprintf("expected stage %s image %q built image info (image name = %s) to be set!", stg.Name(), img.GetName(), stg.GetImage().Name()))
		}

		// Add managed image record only if there was at least one newly built stage
		phase.ShouldAddManagedImageRecord = true

		return nil
	}
}

func (phase *BuildPhase) fetchBaseImageForStage(img *Image, stg stage.Interface) error {
	if stg.Name() == "from" {
		if err := img.FetchBaseImage(phase.Conveyor); err != nil {
			return fmt.Errorf("unable to fetch base image %s for stage %s: %s", img.GetBaseImage().Name(), stg.LogDetailedName(), err)
		}
	} else {
		if freshImgInfo, err := phase.Conveyor.StagesStorage.GetImageInfo(phase.Conveyor.projectName(), phase.StagesIterator.PrevBuiltStage.GetImage().GetStagesStorageImageInfo().Signature, phase.StagesIterator.PrevBuiltStage.GetImage().GetStagesStorageImageInfo().UniqueID); err != nil {
			return err
		} else if freshImgInfo == nil {
			// Reset stages storage cache for this stage
			logboek.Info.LogF("Stages storage cache for %s signature %s is not valid: resetting stages storage cache\n", phase.StagesIterator.PrevBuiltStage.LogDetailedName(), phase.StagesIterator.PrevBuiltStage.GetSignature())
			panic("no")
		}

		return fetchStage(phase.Conveyor.StagesStorage, phase.StagesIterator.PrevBuiltStage)
	}

	return nil
}

func (phase *BuildPhase) calculateStage(img *Image, stg stage.Interface) error {
	stageDependencies, err := stg.GetDependencies(phase.Conveyor, phase.StagesIterator.GetPrevImage(img, stg), phase.StagesIterator.GetPrevBuiltImage(img, stg))
	if err != nil {
		return err
	}

	stageSig, err := calculateSignature(string(stg.Name()), stageDependencies, phase.StagesIterator.PrevNonEmptyStage, phase.Conveyor)
	if err != nil {
		return err
	}
	stg.SetSignature(stageSig)

	var i *container_runtime.StageImage
	var suitableImageFound bool

	cacheExists, cacheImagesDescs, err := getImagesBySignatureFromCache(phase.Conveyor, string(stg.Name()), stageSig)
	if err != nil {
		return err
	}

	if cacheExists {
		if imgInfo, err := selectSuitableStagesStorageImage(stg, cacheImagesDescs); err != nil {
			return err
		} else if imgInfo != nil {
			suitableImageFound = true
			i = phase.Conveyor.GetOrCreateStageImage(phase.StagesIterator.GetPrevImage(img, stg).(*container_runtime.StageImage), imgInfo.Name)
			i.SetStagesStorageImageInfo(imgInfo)
			stg.SetImage(i)
		}
	} else {
		logboek.Info.LogF(
			"Stage %q cache by signature %s is not exists in stages storage cache: resetting stages storage cache\n",
			stg.Name(), stageSig,
		)

		imagesDescs, err := atomicGetImagesBySignatureFromStagesStorageWithCacheReset(phase.Conveyor, stg.LogDetailedName(), stageSig)
		if err != nil {
			return err
		}

		if imgInfo, err := selectSuitableStagesStorageImage(stg, imagesDescs); err != nil {
			return err
		} else if imgInfo != nil {
			suitableImageFound = true

			i = phase.Conveyor.GetOrCreateStageImage(phase.StagesIterator.GetPrevImage(img, stg).(*container_runtime.StageImage), imgInfo.Name)
			i.SetStagesStorageImageInfo(imgInfo)
			stg.SetImage(i)
		}
	}

	if !suitableImageFound {
		// Will build a new image
		i = phase.Conveyor.GetOrCreateStageImage(phase.StagesIterator.GetPrevImage(img, stg).(*container_runtime.StageImage), uuid.New().String())
		stg.SetImage(i)
	}

	if err = stg.AfterSignatureCalculated(phase.Conveyor); err != nil {
		return err
	}

	return nil
}

func (phase *BuildPhase) prepareStageInstructions(img *Image, stg stage.Interface) error {
	logboek.Debug.LogF("-- BuildPhase.prepareStage %s %s\n", img.LogDetailedName(), stg.LogDetailedName())

	stageImage := stg.GetImage()

	serviceLabels := map[string]string{
		imagePkg.WerfDockerImageName:     stageImage.Name(),
		imagePkg.WerfLabel:               phase.Conveyor.projectName(),
		imagePkg.WerfVersionLabel:        werf.Version,
		imagePkg.WerfCacheVersionLabel:   imagePkg.BuildCacheVersion,
		imagePkg.WerfImageLabel:          "false",
		imagePkg.WerfStageSignatureLabel: stg.GetSignature(),
	}

	switch stg.(type) {
	case *stage.DockerfileStage:
		var buildArgs []string

		for key, value := range serviceLabels {
			buildArgs = append(buildArgs, fmt.Sprintf("--label=%s=%s", key, value))
		}

		stageImage.DockerfileImageBuilder().AppendBuildArgs(buildArgs...)

		phase.Conveyor.AppendOnTerminateFunc(func() error {
			return stageImage.DockerfileImageBuilder().Cleanup()
		})

	default:
		imageServiceCommitChangeOptions := stageImage.Container().ServiceCommitChangeOptions()
		imageServiceCommitChangeOptions.AddLabel(serviceLabels)

		if phase.Conveyor.sshAuthSock != "" {
			imageRunOptions := stageImage.Container().RunOptions()
			imageRunOptions.AddVolume(fmt.Sprintf("%s:/.werf/tmp/ssh-auth-sock", phase.Conveyor.sshAuthSock))
			imageRunOptions.AddEnv(map[string]string{"SSH_AUTH_SOCK": "/.werf/tmp/ssh-auth-sock"})
		}
	}

	err := stg.PrepareImage(phase.Conveyor, phase.StagesIterator.GetPrevBuiltImage(img, stg), stageImage)
	if err != nil {
		return fmt.Errorf("error preparing stage %s: %s", stg.Name(), err)
	}

	return nil
}

func (phase *BuildPhase) buildStage(img *Image, stg stage.Interface) error {
	_, err := stapel.GetOrCreateContainer()
	if err != nil {
		return fmt.Errorf("get or create stapel container failed: %s", err)
	}

	infoSectionFunc := func(err error) {
		if err != nil {
			_ = logboek.WithIndent(func() error {
				logImageCommands(stg.GetImage())
				return nil
			})
			return
		}
		logImageInfo(stg.GetImage(), phase.getPrevNonEmptyStageImageSize(), false)
	}

	if err := logboek.Default.LogProcess(
		fmt.Sprintf("Building stage %s", stg.LogDetailedName()),
		logboek.LevelLogProcessOptions{
			InfoSectionFunc: infoSectionFunc,
			Style:           logboek.HighlightStyle(),
		},
		func() (err error) {
			if err := stg.PreRunHook(phase.Conveyor); err != nil {
				return fmt.Errorf("%s preRunHook failed: %s", stg.LogDetailedName(), err)
			}

			return phase.atomicBuildStageImage(img, stg)
		},
	); err != nil {
		return err
	}

	if phase.IntrospectOptions.ImageStageShouldBeIntrospected(img.GetName(), string(stg.Name())) {
		if err := introspectStage(stg); err != nil {
			return err
		}
	}

	return nil
}

func (phase *BuildPhase) atomicBuildStageImage(img *Image, stg stage.Interface) error {
	stageImage := stg.GetImage()

	if err := logboek.WithTag(fmt.Sprintf("%s/%s", img.LogName(), stg.Name()), img.LogTagStyle(), func() error {
		if err := stageImage.Build(phase.ImageBuildOptions); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to build image for stage %s with signature %s: %s", stg.Name(), stg.GetSignature(), err)
	}

	if err := phase.Conveyor.StorageLockManager.LockStage(phase.Conveyor.projectName(), stg.GetSignature()); err != nil {
		return fmt.Errorf("unable to lock project %s signature %s: %s", phase.Conveyor.projectName(), stg.GetSignature(), err)
	}
	defer phase.Conveyor.StorageLockManager.UnlockStage(phase.Conveyor.projectName(), stg.GetSignature())

	imagesDescs, err := atomicGetImagesBySignatureFromStagesStorageWithCacheReset(phase.Conveyor, string(stg.Name()), stg.GetSignature())
	if err != nil {
		return err
	}

	if len(imagesDescs) > 0 {
		var imgInfo *image.Info
		if err := logboek.Info.LogProcess(
			fmt.Sprintf("Selecting suitable image for stage %s by signature %s", stg.Name(), stg.GetSignature()),
			logboek.LevelLogProcessOptions{},
			func() error {
				imgInfo, err = stg.SelectCacheImage(imagesDescs)
				return err
			},
		); err != nil {
			return err
		}

		if imgInfo != nil {
			logboek.Default.LogF(
				"Discarding newly built image for stage %s by signature %s: detected already existing image %s in the stages storage\n",
				stg.Name(), stg.GetSignature(), imgInfo.Name,
			)
			i := phase.Conveyor.GetOrCreateStageImage(phase.StagesIterator.GetPrevImage(img, stg).(*container_runtime.StageImage), imgInfo.Name)
			i.SetStagesStorageImageInfo(imgInfo)
			stg.SetImage(i)

			return nil
		}
	}

	newStageImageName, uniqueID := phase.generateUniqueImageName(stg.GetSignature(), imagesDescs)
	repository, tag := image.ParseRepositoryAndTag(newStageImageName)

	stageImageObj := phase.Conveyor.GetStageImage(stageImage.Name())
	phase.Conveyor.UnsetStageImage(stageImageObj.Name())

	stageImageObj.SetName(newStageImageName)
	stageImageObj.GetStagesStorageImageInfo().Name = newStageImageName
	stageImageObj.GetStagesStorageImageInfo().Repository = repository
	stageImageObj.GetStagesStorageImageInfo().Tag = tag
	stageImageObj.GetStagesStorageImageInfo().Signature = stg.GetSignature()
	stageImageObj.GetStagesStorageImageInfo().UniqueID = uniqueID

	phase.Conveyor.SetStageImage(stageImageObj)

	if err := logboek.Default.LogProcess(
		fmt.Sprintf("Store into stages storage"),
		logboek.LevelLogProcessOptions{},
		func() error {
			if err := phase.Conveyor.StagesStorage.StoreImage(&container_runtime.DockerImage{Image: stageImage}); err != nil {
				return fmt.Errorf("unable to store stage %s signature %s image %s into stages storage %s: %s", stg.LogDetailedName(), stg.GetSignature(), stageImage.Name(), phase.Conveyor.StagesStorage.String(), err)
			}
			return nil
		},
	); err != nil {
		return err
	}

	imagesDescs = append(imagesDescs, stageImage.GetStagesStorageImageInfo())
	return atomicStoreStageCache(phase.Conveyor, string(stg.Name()), stg.GetSignature(), imagesDescs)
}

func (phase *BuildPhase) generateUniqueImageName(signature string, imagesDescs []*image.Info) (string, string) {
	var imageName string

	for {
		timeNow := time.Now().UTC()
		timeNowMicroseconds := timeNow.Unix()*1000 + int64(timeNow.Nanosecond()/1000000)
		uniqueID := fmt.Sprintf("%d", timeNowMicroseconds)
		imageName = phase.Conveyor.StagesStorage.ConstructStageImageName(phase.Conveyor.projectName(), signature, uniqueID)

		for _, imgInfo := range imagesDescs {
			if imgInfo.Name == imageName {
				continue
			}
		}
		return imageName, uniqueID
	}
}

func introspectStage(s stage.Interface) error {
	return logboek.Info.LogProcess(
		fmt.Sprintf("Introspecting stage %s", s.Name()),
		logboek.LevelLogProcessOptions{Style: logboek.HighlightStyle()},
		func() error {
			if err := logboek.WithRawStreamsOutputModeOn(s.GetImage().Introspect); err != nil {
				return fmt.Errorf("introspect error failed: %s", err)
			}

			return nil
		},
	)
}

var (
	logImageInfoLeftPartWidth = 12
	logImageInfoFormat        = fmt.Sprintf("  %%%ds: %%s\n", logImageInfoLeftPartWidth)
)

func logImageInfo(img container_runtime.ImageInterface, prevStageImageSize int64, isUsingCache bool) {
	repository, tag := image.ParseRepositoryAndTag(img.Name())
	logboek.Default.LogFDetails(logImageInfoFormat, "repository", repository)
	logboek.Default.LogFDetails(logImageInfoFormat, "image_id", stringid.TruncateID(img.GetStagesStorageImageInfo().ID))
	logboek.Default.LogFDetails(logImageInfoFormat, "created", img.GetStagesStorageImageInfo().GetCreatedAt())
	logboek.Default.LogFDetails(logImageInfoFormat, "tag", tag)

	if prevStageImageSize == 0 {
		logboek.Default.LogFDetails(logImageInfoFormat, "size", byteCountBinary(img.GetStagesStorageImageInfo().Size))
	} else {
		logboek.Default.LogFDetails(logImageInfoFormat, "diff", byteCountBinary(img.GetStagesStorageImageInfo().Size-prevStageImageSize))
	}

	if !isUsingCache {
		changes := img.Container().UserCommitChanges()
		if len(changes) != 0 {
			fitTextOptions := logboek.FitTextOptions{ExtraIndentWidth: logImageInfoLeftPartWidth + 4}
			formattedCommands := strings.TrimLeft(logboek.FitText(strings.Join(changes, "\n"), fitTextOptions), " ")
			logboek.Default.LogFDetails(logImageInfoFormat, "instructions", formattedCommands)
		}

		logImageCommands(img)
	}
}

func logImageCommands(img container_runtime.ImageInterface) {
	commands := img.Container().UserRunCommands()
	if len(commands) != 0 {
		fitTextOptions := logboek.FitTextOptions{ExtraIndentWidth: logImageInfoLeftPartWidth + 4}
		formattedCommands := strings.TrimLeft(logboek.FitText(strings.Join(commands, "\n"), fitTextOptions), " ")
		logboek.Default.LogFDetails(logImageInfoFormat, "commands", formattedCommands)
	}
}

func byteCountBinary(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func calculateSignature(stageName, stageDependencies string, prevNonEmptyStage stage.Interface, conveyor *Conveyor) (string, error) {
	checksumArgs := []string{image.BuildCacheVersion, stageName, stageDependencies}
	if prevNonEmptyStage != nil {
		prevStageDependencies, err := prevNonEmptyStage.GetNextStageDependencies(conveyor)
		if err != nil {
			return "", fmt.Errorf("unable to get prev stage %s dependencies for the stage %s: %s", prevNonEmptyStage.Name(), stageName, err)
		}

		checksumArgs = append(checksumArgs, prevNonEmptyStage.GetSignature(), prevStageDependencies)
	}

	signature := util.Sha3_224Hash(checksumArgs...)

	blockMsg := fmt.Sprintf("Stage %s signature %s", stageName, signature)
	_ = logboek.Debug.LogBlock(blockMsg, logboek.LevelLogBlockOptions{}, func() error {
		checksumArgsNames := []string{
			"BuildCacheVersion",
			"stageName",
			"stageDependencies",
			"prevNonEmptyStage signature",
			"prevNonEmptyStage dependencies for next stage",
		}
		for ind, checksumArg := range checksumArgs {
			logboek.Debug.LogF("%s => %q\n", checksumArgsNames[ind], checksumArg)
		}
		return nil
	})

	return signature, nil
}
