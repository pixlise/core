package main

var usersIdsToIgnore = map[string]string{
	"5e3b3bc480ee5c191714d6b7": "Tom Barber",
	"6386b4e01ae980e2dc3bbee4": "Test User - Integration test login",
	"600f2a0806b6c70071d3d174": "TEST USER",
	"5f45d7b8b5abff006d4fdb91": "TEST USER - For User Management Role Unit Test",
	"5ee0258dba28c3001931719f": "Henry Jiao",
	"5f47efd2b5abff006d501804": "henrygjiao@gmail.com",
	"645adaef68940b204da217ea": "snowcrazed@gmail.com",
	"645d2e411d4c052b590b7ad6": "ryanastonebraker@gmail.com",
	"6095863e58f57300728a45f2": "scottdavidoff@gmail.com",
	/*"5de45d85ca40070f421a3a34": "Peter N",
	"6227d96292150a0069117483": "Ryan JPL",
	"61dc92ada856070069220afd": "Michael Fedell",
	"5df31aef08f9630ec08ada4e": "Scott Davidoff JPL",*/
}

func shouldIgnoreUser(id string) bool {
	for ignoreId := range usersIdsToIgnore {
		if id == ignoreId {
			return true
		}
	}
	return false
}
