package wsHelpers

import (
	"fmt"
	"os"
)

func ResetLocalMongoBackupDir() error {
	os.RemoveAll("./backup")
	err := os.Mkdir("./backup", 0750)
	if err != nil {
		return fmt.Errorf("PIXLISE Backup failed to create backup directory: %v", err)
	}

	return nil
}

func ClearLocalMongoArchive() error {
	err := os.RemoveAll(dataBackupLocalPath)
	if err != nil {
		return fmt.Errorf("Failed to remove local DB archive files in: %v. Error; %v", dataBackupLocalPath, err)
	}

	return nil
}
