package stages_manager

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/flant/shluz"

	"github.com/flant/werf/pkg/image"

	"github.com/flant/werf/pkg/werf"

	"github.com/flant/logboek"
	"github.com/flant/werf/pkg/build/stage"
	"github.com/flant/werf/pkg/container_runtime"
	"github.com/flant/werf/pkg/storage"
	"gopkg.in/yaml.v2"
)

type StagesManager struct {
	StagesStorageByProjectDir string
	ProjectName               string

	StorageLockManager storage.LockManager
	StagesStorage      storage.StagesStorage
	StagesStorageCache storage.StagesStorageCache
}

func NewStagesManager(projectName string, storageLockManager storage.LockManager, stagesStorageCache storage.StagesStorageCache) *StagesManager {
	return &StagesManager{
		StagesStorageByProjectDir: filepath.Join(werf.GetServiceDir(), "stages_storage_by_project"),
		ProjectName:               projectName,
		StorageLockManager:        storageLockManager,
		StagesStorageCache:        stagesStorageCache,
	}
}

//func (m *StagesManager) GetAllStagesIDs() ([]image.StageID, error) {
//if cacheExists, images, err := m.StagesStorageCache.GetAllStages(m.ProjectName); err != nil {
//	return nil, err
//} else if cacheExists {
//	return images, nil
//} else {
//return m.StagesStorage.GetAllStages(m.ProjectName)
//}
//}

func (m *StagesManager) readCurrentProjectStagesStorageAddress() (string, error) {
	f := filepath.Join(m.StagesStorageByProjectDir, m.ProjectName)
	if _, err := os.Stat(f); os.IsNotExist(err) {
		return "", nil
	} else if err != nil {
		return "", fmt.Errorf("error accessing %s: %s", f, err)
	}

	if dataBytes, err := ioutil.ReadFile(f); err != nil {
		return "", fmt.Errorf("error reading %s: %s", f, err)
	} else {
		return strings.TrimSpace(string(dataBytes)), nil
	}
}

func (m *StagesManager) checkProjectStagesStorageNotChanged(stagesStorageAddress string) error {
	if currentStagesStorageAddress, err := m.readCurrentProjectStagesStorageAddress(); err != nil {
		return err
	} else if currentStagesStorageAddress != stagesStorageAddress {
		return fmt.Errorf(
			`Project %q already uses another stages storage %q!
Run the following command to move existing project stages to the new stages storage:
'werf stages mv --from-stages-storage=%s --to-stages-storage=%s'

Or simply switch project to the new stages storage by the following command:
'werf stages switch -s %s'`,
			m.ProjectName, currentStagesStorageAddress, currentStagesStorageAddress, stagesStorageAddress, stagesStorageAddress)
	}

	return nil
}

func (m *StagesManager) writeProjectStagesStorage(stagesStorageAddress string) error {
	f := filepath.Join(m.StagesStorageByProjectDir, m.ProjectName)
	d := filepath.Dir(f)
	if err := os.MkdirAll(d, os.ModePerm); err != nil {
		return fmt.Errorf("error creating dir %s: %s", d, err)
	}
	if err := ioutil.WriteFile(f, []byte(fmt.Sprintf("%s\n", stagesStorageAddress)), 0644); err != nil {
		return fmt.Errorf("error writing %s: %s", f, err)
	}
	return nil
}

func (m *StagesManager) SwitchStagesStorage(newStagesStorage storage.StagesStorage) error {
	lockName := fmt.Sprintf("stages_storage_by_project.%s", m.ProjectName)
	if err := shluz.Lock(lockName, shluz.LockOptions{}); err != nil {
		return err
	}
	defer shluz.Unlock(lockName)

	if currentStagesStorageAddress, err := m.readCurrentProjectStagesStorageAddress(); err != nil {
		return err
	} else if currentStagesStorageAddress != "" {
		if currentStagesStorageAddress == newStagesStorage.Address() {
			logboek.Default.LogFDetails("Stages storage not changed — %s\n", currentStagesStorageAddress)
			m.StagesStorage = newStagesStorage
			return nil
		} else {
			logboek.Default.LogFDetails("Old stages storage — %s\n", currentStagesStorageAddress)
		}
	}
	logboek.Default.LogFDetails("New stages storage — %s\n", newStagesStorage.Address())

	if err := m.writeProjectStagesStorage(newStagesStorage.Address()); err != nil {
		return err
	}
	m.StagesStorage = newStagesStorage
	return nil
}

func (m *StagesManager) UseStagesStorage(stagesStorage storage.StagesStorage) error {
	f := filepath.Join(m.StagesStorageByProjectDir, m.ProjectName)
	if _, err := os.Stat(f); os.IsNotExist(err) {
		lockName := fmt.Sprintf("stages_storage_by_project.%s", m.ProjectName)
		if err := shluz.Lock(lockName, shluz.LockOptions{}); err != nil {
			return err
		}
		defer shluz.Unlock(lockName)

		if _, err := os.Stat(f); os.IsNotExist(err) {
			if err := m.writeProjectStagesStorage(stagesStorage.Address()); err != nil {
				return err
			}
			m.StagesStorage = stagesStorage
			return nil
		} else if err != nil {
			return fmt.Errorf("error accessing %s: %s", f, err)
		} else {
			if err := m.checkProjectStagesStorageNotChanged(stagesStorage.Address()); err != nil {
				return err
			}
			m.StagesStorage = stagesStorage
			return nil
		}
	} else if err != nil {
		return fmt.Errorf("error accessing %s: %s", f, err)
	} else {
		if err := m.checkProjectStagesStorageNotChanged(stagesStorage.Address()); err != nil {
			return err
		}
		m.StagesStorage = stagesStorage
		return nil
	}
}

func (m *StagesManager) GetAllStages() ([]*image.StageDescription, error) {
	// TODO: optimize, get list from coherent stages storage cache
	if stageIDs, err := m.StagesStorage.GetAllStages(m.ProjectName); err != nil {
		return nil, err
	} else {
		var stages []*image.StageDescription

		for _, stageID := range stageIDs {
			if stageDesc, err := m.getStageDescription(stageID); err != nil {
				return nil, err
			} else if stageDesc == nil {
				return nil, fmt.Errorf("invalid stage %s: stage does not exists in the %s", stageID.String(), m.StagesStorage.String())
			} else {
				stages = append(stages, stageDesc)
			}
		}

		return stages, nil
	}
}

func (m *StagesManager) DeleteStages(options storage.DeleteImageOptions, stages ...*image.StageDescription) error {
	for _, stageDesc := range stages {
		if err := m.StagesStorageCache.DeleteStagesBySignature(m.ProjectName, stageDesc.StageID.Signature); err != nil {
			return fmt.Errorf("unable to delete %s %s stages storage cache record: %s", err)
		}
	}
	return m.StagesStorage.DeleteStages(options, stages...)
}

func (m *StagesManager) FetchStage(stg stage.Interface) error {
	if freshStageDescription, err := m.StagesStorage.GetStageDescription(m.ProjectName, stg.GetImage().GetStageDescription().StageID.Signature, stg.GetImage().GetStageDescription().StageID.UniqueID); err != nil {
		return err
	} else if freshStageDescription == nil {
		// TODO: stages manager should report to the conveyor that conveoyor should be reset
		return fmt.Errorf("Invalid stage %s image %q! Stage is no longer available in the %s. Remove cache directory %s and retry!", stg.LogDetailedName(), stg.GetImage().Name(), m.StagesStorage.String(), filepath.Join(werf.GetLocalCacheDir(), "stages_storage_v4", m.ProjectName, stg.GetSignature()))
	}

	if shouldFetch, err := m.StagesStorage.ShouldFetchImage(&container_runtime.DockerImage{Image: stg.GetImage()}); err == nil && shouldFetch {
		if err := logboek.Default.LogProcess(
			fmt.Sprintf("Fetching stage %s from stages storage", stg.LogDetailedName()),
			logboek.LevelLogProcessOptions{Style: logboek.HighlightStyle()},
			func() error {
				logboek.Info.LogF("Image name: %s\n", stg.GetImage().Name())
				if err := m.StagesStorage.FetchImage(&container_runtime.DockerImage{Image: stg.GetImage()}); err != nil {
					return fmt.Errorf("unable to fetch stage %s image %s from stages storage %s: %s", stg.LogDetailedName(), stg.GetImage().Name(), m.StagesStorage.String(), err)
				}
				return nil
			},
		); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	return nil
}

func (m *StagesManager) SelectSuitableStage(stg stage.Interface, stages []*image.StageDescription) (*image.StageDescription, error) {
	if len(stages) == 0 {
		return nil, nil
	}

	var stageDesc *image.StageDescription
	if err := logboek.Info.LogProcess(
		fmt.Sprintf("Selecting suitable image for stage %s by signature %s", stg.Name(), stg.GetSignature()),
		logboek.LevelLogProcessOptions{},
		func() error {
			var err error
			stageDesc, err = stg.SelectSuitableStage(stages)
			return err
		},
	); err != nil {
		return nil, err
	}
	if stageDesc == nil {
		return nil, nil
	}

	imgInfoData, err := yaml.Marshal(stageDesc)
	if err != nil {
		panic(err)
	}

	_ = logboek.Debug.LogBlock("Selected cache image", logboek.LevelLogBlockOptions{Style: logboek.HighlightStyle()}, func() error {
		logboek.Debug.LogF(string(imgInfoData))
		return nil
	})

	return stageDesc, nil
}

func (m *StagesManager) AtomicStoreStagesBySignatureToCache(stageName, stageSig string, stageIDs []image.StageID) error {
	if err := m.StorageLockManager.LockStageCache(m.ProjectName, stageSig); err != nil {
		return fmt.Errorf("error locking stage %s cache by signature %s: %s", stageName, stageSig, err)
	}
	defer m.StorageLockManager.UnlockStageCache(m.ProjectName, stageSig)

	return logboek.Info.LogProcess(
		fmt.Sprintf("Storing stage %s images by signature %s into stages storage cache", stageName, stageSig),
		logboek.LevelLogProcessOptions{},
		func() error {
			if err := m.StagesStorageCache.StoreStagesBySignature(m.ProjectName, stageSig, stageIDs); err != nil {
				return fmt.Errorf("error storing stage %s images by signature %s into stages storage cache: %s", stageName, stageSig, err)
			}
			return nil
		},
	)
}

func (m *StagesManager) GetStagesBySignature(stageName, stageSig string) ([]*image.StageDescription, error) {
	cacheExists, cacheStages, err := m.getStagesBySignatureFromCache(stageName, stageSig)
	if err != nil {
		return nil, err
	}
	if cacheExists {
		return cacheStages, nil
	}

	logboek.Info.LogF(
		"Stage %s cache by signature %s is not exists in the stages storage cache: resetting stages storage cache\n",
		stageName, stageSig,
	)
	return m.atomicGetStagesBySignatureWithCacheReset(stageName, stageSig)
}

func (m *StagesManager) getStagesBySignatureFromCache(stageName, stageSig string) (bool, []*image.StageDescription, error) {
	var cacheExists bool
	var cacheStagesIDs []image.StageID

	err := logboek.Info.LogProcess(
		fmt.Sprintf("Getting stage %s images by signature %s from stages storage cache", stageName, stageSig),
		logboek.LevelLogProcessOptions{},
		func() error {
			var err error
			cacheExists, cacheStagesIDs, err = m.StagesStorageCache.GetStagesBySignature(m.ProjectName, stageSig)
			if err != nil {
				return fmt.Errorf("error getting project %s stage %s images from stages storage cache: %s", m.ProjectName, stageSig, err)
			}
			return nil
		},
	)

	var stages []*image.StageDescription

	for _, stageID := range cacheStagesIDs {
		if stageDesc, err := m.getStageDescription(stageID); err != nil {
			return false, nil, err
		} else {
			stages = append(stages, stageDesc)
		}
	}

	return cacheExists, stages, err
}

func (m *StagesManager) atomicGetStagesBySignatureWithCacheReset(stageName, stageSig string) ([]*image.StageDescription, error) {
	if err := m.StorageLockManager.LockStageCache(m.ProjectName, stageSig); err != nil {
		return nil, fmt.Errorf("error locking project %s stage %s cache: %s", m.ProjectName, stageSig, err)
	}
	defer m.StorageLockManager.UnlockStageCache(m.ProjectName, stageSig)

	var stageIDs []image.StageID
	if err := logboek.Default.LogProcess(
		fmt.Sprintf("Get %s stages by signature %s from stages storage", stageName, stageSig),
		logboek.LevelLogProcessOptions{},
		func() error {
			var err error
			stageIDs, err = m.StagesStorage.GetStagesBySignature(m.ProjectName, stageSig)
			if err != nil {
				return fmt.Errorf("error getting project %s stage %s images from stages storage: %s", m.StagesStorage.String(), stageSig, err)
			}

			logboek.Debug.LogF("Stages ids: %#v\n", stageIDs)

			return nil
		},
	); err != nil {
		return nil, err
	}

	var stages []*image.StageDescription
	for _, stageID := range stageIDs {
		if stageDesc, err := m.getStageDescription(stageID); err != nil {
			return nil, err
		} else {
			stages = append(stages, stageDesc)
		}
	}

	if err := logboek.Info.LogProcess(
		fmt.Sprintf("Storing %s stages by signature %s into stages storage cache", stageName, stageSig),
		logboek.LevelLogProcessOptions{},
		func() error {
			if err := m.StagesStorageCache.StoreStagesBySignature(m.ProjectName, stageSig, stageIDs); err != nil {
				return fmt.Errorf("error storing stage %s images by signature %s into stages storage cache: %s", stageName, stageSig, err)
			}
			return nil
		},
	); err != nil {
		return nil, err
	}

	return stages, nil
}

func (m *StagesManager) getStageDescription(stageID image.StageID) (*image.StageDescription, error) {
	stageImageName := m.StagesStorage.ConstructStageImageName(m.ProjectName, stageID.Signature, stageID.UniqueID)

	logboek.Info.LogF("Getting image %s info from manifest cache...\n", stageImageName)
	if imgInfo, err := image.CommonManifestCache.GetImageInfo(stageImageName); err != nil {
		return nil, fmt.Errorf("error getting image %s info from manifest cache: %s", stageImageName, err)
	} else if imgInfo != nil {
		logboek.Info.LogF("Got image %s info from manifest cache (CACHE HIT)\n", stageImageName)
		return &image.StageDescription{
			StageID: &image.StageID{Signature: stageID.Signature, UniqueID: stageID.UniqueID},
			Info:    imgInfo,
		}, nil
	} else {
		logboek.Info.LogF("Not found %s image info in manifest cache (CACHE MISS)\n", stageImageName)
		logboek.Info.LogF("Getting signature %q uniqueID %q stage info from %s...\n", stageID.Signature, stageID.UniqueID, m.StagesStorage.String())
		if stageDesc, err := m.StagesStorage.GetStageDescription(m.ProjectName, stageID.Signature, stageID.UniqueID); err != nil {
			return nil, fmt.Errorf("error getting signature %q uniqueID %q stage info from %s: %s", stageID.Signature, stageID.UniqueID, m.StagesStorage.String(), err)
		} else if stageDesc != nil {
			logboek.Info.LogF("Storing image %s info into manifest cache\n", stageImageName)
			if err := image.CommonManifestCache.StoreImageInfo(stageDesc.Info); err != nil {
				return nil, fmt.Errorf("error storing image %s info into manifest cache: %s", stageImageName, err)
			}
			return stageDesc, nil
		} else {
			logboek.Default.LogF("Not found signature %q uniqueID %q stage info in %s\n", stageID.Signature, stageID.UniqueID, m.StagesStorage.String())
			return nil, fmt.Errorf("Invalid stage by signature %q uniqueID %q found in the stages storage cache! Stage is no longer available in the %s. Remove cache directory %s and retry!", stageID.Signature, stageID.UniqueID, m.StagesStorage.String(), filepath.Join(werf.GetLocalCacheDir(), "stages_storage_v4", m.ProjectName))
		}
	}
}

func (m *StagesManager) GenerateStageUniqueID(signature string, stages []*image.StageDescription) (string, string) {
	var imageName string

	for {
		timeNow := time.Now().UTC()
		timeNowMicroseconds := timeNow.Unix()*1000 + int64(timeNow.Nanosecond()/1000000)
		uniqueID := fmt.Sprintf("%d", timeNowMicroseconds)
		imageName = m.StagesStorage.ConstructStageImageName(m.ProjectName, signature, uniqueID)

		for _, stageDesc := range stages {
			if stageDesc.Info.Name == imageName {
				continue
			}
		}
		return imageName, uniqueID
	}
}
