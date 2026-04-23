package mongoDBConnection

import (
	"encoding/json"
	"fmt"
)

func Example_mongoDBConnection_read() {
	var info MongoConnectionInfo
	secretValue := `{"dbClusterIdentifier":"pixlise-db","password":"p@ssword","engine":"mongo","port":"27017","host":"172.31.66.202","ssl":"false","username":"us3r"}`
	err := json.Unmarshal([]byte(secretValue), &info)
	fmt.Printf("%v|%v\n", err, info)

	// Output:
	// <nil>|{172.31.66.202 us3r p@ssword }
}
