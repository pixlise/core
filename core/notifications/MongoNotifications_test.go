package notifications

//func Example_setNotifications(){
//
//	m := MongoUtils{}
//
//	//r, err  := m.GetAllMongoUsers(logger.NullLogger{})
//	//if err != nil{
//	//	fmt.Printf("%v", err)
//	//}
//	//fmt.Printf("%v", r)
//
//	r, err := m.GetMongoSubscribersByTopicID([]string{"5df31aef08f9630ec08ada4e"}, "user-quant-complete", logger.NullLogger{})
//	if err != nil{
//		fmt.Printf("%v", err)
//	}
//	fmt.Printf("%v", r)
//	// Output:
//	//
//}
//
//func Example_getNotifications(){
//
//	test := NotificationStack{
//		Notifications: nil,
//		FS:            nil,
//		Bucket:        "",
//		Track:         nil,
//		AdminEmails:   nil,
//		Environment:   "",
//		Logger:        nil,
//		Backend:       "MONGO",
//	}
//
//	notes, err := test.GetUINotifications("sample-user")
//	if err != nil{
//		fmt.Printf("%v", err)
//	}
//	for _, n := range notes {
//		fmt.Printf("%v", n)
//	}
//
//	// Output:
//	//
//}
