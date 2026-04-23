package mongoDBConnection

import "fmt"

func Example_mongoDBConnection_MakeMongoURI() {
	fmt.Println(MakeMongoURI("", ""))
	fmt.Println(MakeMongoURI("mongodb://localhost", ""))
	fmt.Println(MakeMongoURI("localhost", ""))
	fmt.Println(MakeMongoURI("mongodb://localhost", "readSecondary=true"))
	fmt.Println(MakeMongoURI("192.168.1.10:27017", "?replicaSet=rs0&readPreference=secondary"))
	fmt.Println(MakeMongoURI("192.168.1.10:27017", "/replicaSet=rs0&readPreference=secondary"))
	fmt.Println(MakeMongoURI("192.168.1.10:27017", "/?replicaSet=rs0&readPreference=secondary"))
	fmt.Println(MakeMongoURI("mongodb://192.168.1.10:27017", "replicaSet=rs0&readPreference=secondary"))
	fmt.Println(MakeMongoURI("mongodb://192.168.1.10:27017/", "replicaSet=rs0&readPreference=secondary"))
	fmt.Println(MakeMongoURI("mongodb://192.168.1.10:27017/", "replicaSet=rs0&"))
	fmt.Println(MakeMongoURI("mongodb://192.168.1.10:27017/", "/?replicaSet=rs0&readPreference=secondary"))

	// Output:
	// mongodb://localhost
	// mongodb://localhost
	// mongodb://localhost
	// mongodb://localhost/?readSecondary=true
	// mongodb://192.168.1.10:27017/?replicaSet=rs0&readPreference=secondary
	// mongodb://192.168.1.10:27017/?replicaSet=rs0&readPreference=secondary
	// mongodb://192.168.1.10:27017/?replicaSet=rs0&readPreference=secondary
	// mongodb://192.168.1.10:27017/?replicaSet=rs0&readPreference=secondary
	// mongodb://192.168.1.10:27017/?replicaSet=rs0&readPreference=secondary
	// mongodb://192.168.1.10:27017/?replicaSet=rs0
	// mongodb://192.168.1.10:27017/?replicaSet=rs0&readPreference=secondary
}
