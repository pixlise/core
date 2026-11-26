package expressionrunner

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/pixlise/core/v4/api/services/servicesMock"
	"github.com/pixlise/core/v4/core/idgen"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/timestamper"
	"github.com/pixlise/core/v4/core/wstestlib"
)

func Example_expressionrunner_FetchSourceCode() {
	exprId := "u59sahioy18frfl9"
	scanId := "048300551"
	quantId := "quant-ggy6zxhn23p7rlv9"

	modIds := []string{"idc2d7xifmbpqk8o", "ng46r8vwzr3z28ui", "f6hrn69g5tuyiq3m", "yg7o9dkue0orim26"}
	modVers := []string{"v1.3.0", "v0.8.0", "v0.33.0", "v3.5.5"}

	idGen := idgen.MockIDGenerator{
		IDs: []string{"id123"},
	}
	logLev := logger.LogDebug
	svcs := servicesMock.MakeMockSvcsWithFS("./test-files/", &idGen, &logLev)
	ts := []int64{}
	for c := 0; c < 30; c++ {
		ts = append(ts, int64(1783596971+c))
	}
	svcs.TimeStamper = &timestamper.MockTimeNowStamper{QueuedTimeStamps: ts}

	svcs.MongoDB = wstestlib.GetDBWithEnvironment("backendexpr")

	err := svcs.MongoDB.Drop(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	SeedDBForExpressionTest("../../test-files-db-seed", scanId, quantId, exprId, modIds, modVers, svcs.MongoDB)
	src, expr, err := FetchSourceCode(exprId, scanId, quantId, "user-123", &svcs)

	fmt.Printf("Error: %v\n", err)
	fmt.Printf("Expr: %v, type %v, modules %v\n", expr.Id, expr.SourceLanguage, len(expr.ModuleReferences))

	expSrc, e := os.ReadFile("./test-files/expected-fetched-source.txt")
	if e != nil {
		log.Fatal(e)
	}

	/*
		e := os.WriteFile("./test-files/fetched-source.txt", []byte(src), 0777)
		line := 0
		ex := string(expSrc)
		for i := range ex {
			if src[i] == '\n' {
				line++
			}
			if ex[i] != src[i] {
				fmt.Printf("Differs at line %v, pos %v, src=%v, exp=%v\nsrc=\"%v\"\nexp=\"%v\"\n", line, i, src[i], ex[i], src[i-10:i+10], ex[i-10:i+10])
				break
			}
		}
	*/
	fmt.Printf("Source ok: %v\n", src == string(expSrc))

	// Output:
	// Error: <nil>
	// Expr: u59sahioy18frfl9, type LUA, modules 4
	// Source ok: true
}
