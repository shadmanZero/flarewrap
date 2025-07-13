package storage

import (
	"os"
	"path/filepath"
	"github.com/shadmanZero/flarewrap/internal/image"
	)


var (
	WORKING_DRIS = []string{
		"images",
		"temp",
		"machines",
		"logs",
	}
)
type StorageManager struct {
	WorkingDir string
}

func NewStorageManager(workingDir string) *StorageManager {
	return &StorageManager{
		WorkingDir: workingDir,
	}
}

func (sm *StorageManager) CreateDirectoryStructure() error {
	for _, dir := range WORKING_DRIS {
		dirPath := filepath.Join(sm.WorkingDir, dir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			err := os.MkdirAll(dirPath, 0755)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
func (sm *StorageManager) GetImagePath(image *image.Image) string {
	return filepath.Join(sm.WorkingDir, "images", image.ImageName + ".img")
}
func (sm *StorageManager) GetImageMetadataPath(image *image.Image) string {
	return filepath.Join(sm.WorkingDir, "images", image.ImageName + ".json")	
}
func (sm *StorageManager) IsImageExists(image *image.Image) bool {
	_, err := os.Stat(sm.GetImagePath(image))
	return err == nil
}