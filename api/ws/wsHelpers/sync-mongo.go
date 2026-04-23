package wsHelpers

import (
	"fmt"
	"os"
)

func ResetLocalMongoBackupDir(dataBackupLocalPath string) error {
	err := os.RemoveAll(dataBackupLocalPath)
	if err != nil {
		return fmt.Errorf("PIXLISE Backup failed to RemoveAll backup directory: %v", err)
	}

	err = os.Mkdir(dataBackupLocalPath, 0750)
	if err != nil {
		return fmt.Errorf("PIXLISE Backup failed to create backup directory: %v", err)
	}

	return nil
}

func ClearLocalMongoArchive(dataBackupLocalPath string) error {
	err := os.RemoveAll(dataBackupLocalPath)
	if err != nil {
		return fmt.Errorf("Failed to remove local DB archive files in: %v. Error; %v", dataBackupLocalPath, err)
	}

	return nil
}
