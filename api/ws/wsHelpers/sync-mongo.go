package wsHelpers

import (
	"fmt"
	"os"
	"strings"

	"github.com/mongodb/mongo-tools/common/log"
	"github.com/mongodb/mongo-tools/common/options"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/mongoDBConnection"
)

func ResetLocalMongoBackupDir() error {
	err := os.RemoveAll("./backup")
	if err != nil {
		return fmt.Errorf("PIXLISE Backup failed to RemoveAll backup directory: %v", err)
	}

	err = os.Mkdir("./backup", 0750)
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

func makeMongoToolOptions(mongoDetails mongoDBConnection.MongoConnectionDetails, logger logger.ILogger, dbNamespace string) (*options.ToolOptions, error) {
	var toolOptions *options.ToolOptions

	log.SetVerbosity(nil /*toolOptions.Verbosity*/)
	lw := LogWriter{logger: logger}
	log.SetWriter(lw)

	ssl := options.SSL{}

	isLocal := strings.Contains(mongoDetails.Host, "localhost") && len(mongoDetails.User) <= 0 && len(mongoDetails.Password) <= 0

	if !isLocal {
		ssl = options.SSL{
			UseSSL:        true,
			SSLCAFile:     "./global-bundle.pem",
			SSLPEMKeyFile: "./global-bundle.pem",
		}
	}

	auth := options.Auth{
		Username: mongoDetails.User,
		Password: mongoDetails.Password,
	}

	connection := &options.Connection{
		Host: mongoDetails.Host,
	}

	// Trim excess
	protocolPrefix := "mongodb://"
	connection.Host = strings.TrimPrefix(connection.Host, protocolPrefix)

	connectionURI := fmt.Sprintf("mongodb://%s/%s", connection.Host, "")

	uri, err := options.NewURI(connectionURI)
	if err != nil {
		return nil, err
	}

	retryWrites := false

	toolOptions = &options.ToolOptions{
		RetryWrites: &retryWrites,
		SSL:         &ssl,
		Connection:  connection,
		Auth:        &auth,
		Verbosity:   &options.Verbosity{},
		URI:         uri,
		Namespace:   &options.Namespace{DB: dbNamespace},
	}

	return toolOptions, nil
}
