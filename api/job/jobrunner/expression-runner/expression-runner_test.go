package expressionrunner

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/services/servicesMock"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/idgen"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/timestamper"
	"github.com/pixlise/core/v4/core/wstestlib"
	protos "github.com/pixlise/core/v4/generated-protos"
)

/*func Example_expressionrunner_RunExpression_Simple() {
	err := runSource("print(\"hello\")")
	fmt.Printf("%v\n", err)

	// Output:
	// hello
	// <nil>
}*/

// Short tests all use the Naltsos dataset which is much smaller
// We run long tests using Castle Geyser below
func Test_expressionrunner_RunExpression_ShortTests(t *testing.T) {
	exprId := "u59sahioy18frfl9"
	scanId := "048300551"
	quantId := "quant-ggy6zxhn23p7rlv9"

	modIds := []string{"idc2d7xifmbpqk8o", "ng46r8vwzr3z28ui", "f6hrn69g5tuyiq3m", "yg7o9dkue0orim26"}
	modVers := []string{"v1.3.0", "v0.8.0", "v0.33.0", "v3.5.5"}

	testType := map[string]string{
		"NoMemo":        "",
		"MemoNonQuant":  fmt.Sprintf(".*_geometry_%v.*", scanId),
		"MemoOnlyQuant": fmt.Sprintf(".*_%v_quant-.*", scanId),
		"MemoAll":       fmt.Sprintf(".*%v.*", scanId),
	}

	expectedResult := `RunExpession error: <nil>

Got map size 121

Returned map matches expected output from PIXLISE`

	for name, memoinclude := range testType {
		t.Logf("RunExpression testing scan: %v, expression: %v (%v)", scanId, exprId, name)
		result := runExpressionTest(scanId, quantId, exprId, modIds, modVers, memoinclude)
		//t.Log(result)

		if result != expectedResult {
			t.Errorf("\nRunExpression testing scan: %v, expression: %v (%v) FAILED\nGot result:\n%v\n------\nExpected result:\n%v", scanId, exprId, name, result, expectedResult)
		}
	}
}

// We run long tests using Castle Geyser
func Test_expressionrunner_RunExpression_LongTests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long expression tests in short mode")
		return
	}

	exprIds := []string{"u59sahioy18frfl9", "750idrpn2ql3j4fu"}
	scanId := "393871873"
	quantId := "quant-pvkostn8a2u6j7cj"

	modIds := []string{"idc2d7xifmbpqk8o", "ng46r8vwzr3z28ui", "f6hrn69g5tuyiq3m", "yg7o9dkue0orim26"}
	modVers := []string{"v1.3.0", "v0.8.0", "v0.33.0", "v3.5.5"}

	testType := []map[string]string{
		{
			"NoMemo":        "",
			"MemoNonQuant":  fmt.Sprintf(".*_geometry_%v.*", scanId),
			"MemoOnlyQuant": fmt.Sprintf(".*_%v_quant-.*", scanId),
			"MemoAll":       fmt.Sprintf(".*%v.*", scanId),
		},
		// NOTE: This expression doesn't use quant data!
		{
			"NoMemo":  "",
			"MemoAll": fmt.Sprintf(".*%v.*", scanId),
		},
	}

	expectedResult := `RunExpession error: <nil>

Got map size 3333

Returned map matches expected output from PIXLISE`

	for c, exprId := range exprIds {
		for name, memoinclude := range testType[c] {
			t.Logf("RunExpression testing scan: %v, expression: %v (%v)", scanId, exprId, name)
			result := runExpressionTest(scanId, quantId, exprId, modIds, modVers, memoinclude)
			//t.Log(result)

			if result != expectedResult {
				t.Errorf("\nRunExpression testing scan: %v, expression: %v (%v) FAILED\nGot result:\n%v\n------\nExpected result:\n%v", scanId, exprId, name, result, expectedResult)
			}
		}
	}
}

func runExpressionTest(scanId, quantId, exprId string, modIds, modVers []string, seedMemoInclude string) string {
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

	// If the memo include string is empty, we don't want to seed ANY memoised JSON in DB, otherwise only include if
	// it matches the regex provided
	if len(seedMemoInclude) > 0 {
		fs := fileaccess.FSAccess{}
		memPath := "./test-files/memoisation"
		files, err := fs.ListObjects(memPath, "")
		if err != nil {
			log.Fatal(err)
		}

		for _, f := range files {
			if match, err := regexp.MatchString(seedMemoInclude, f); err == nil && match {
				// Read this into our memoisation table
				err = SeedDB(f[0:len(f)-5] /*.json*/, filepath.Join(memPath, f), dbCollections.MemoisedItemsName, &protos.MemoisedItem{}, svcs.MongoDB)
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}

	m, _, _, err := RunExpression(exprId, scanId, quantId,
		svcs.Log, svcs.MongoDB, svcs.TimeStamper, svcs.FS,
		svcs.Config.ConfigBucket, svcs.Config.UsersBucket, svcs.Config.DatasetsBucket, true, false)
	sz := 0
	if m != nil {
		sz = len(m.Values)
	}

	result := []string{}
	result = append(result, fmt.Sprintf("RunExpession error: %v\n", err))

	if err == nil {
		result = append(result, fmt.Sprintf("Got map size %v\n", sz))

		// Compare with the CSV we got from PIXLISE
		err = compareMapWithCSV(m, fmt.Sprintf("./test-files/PIXLISE_output_%v_%v.csv", scanId, exprId))
		if err != nil {
			result = append(result, fmt.Sprintf("Failed to match output to expected: %v", err))
		} else {
			result = append(result, "Returned map matches expected output from PIXLISE")
		}
	}

	return strings.Join(result, "\n")
}

func compareMapWithCSV(m *PMCDataValues, csvPath string) error {
	csvFile, err := os.Open(csvPath)
	if err != nil {
		return fmt.Errorf("ReadExpectedCSV: %v\n", err)
	}

	r := csv.NewReader(csvFile)
	eM, err := r.ReadAll()
	if err != nil {
		return fmt.Errorf("ReadExpectedCSV2: %v\n", err)
	}

	for c, line := range eM {
		if len(line) != 2 {
			return fmt.Errorf("CSV line %v has wrong number of fields", c)
		}

		pmc, err := strconv.Atoi(line[0])
		if err != nil {
			return fmt.Errorf("CSV line %v has bad pmc: %v", c, line[0])
		}
		value, err := strconv.ParseFloat(line[1], 64)
		if err != nil {
			return fmt.Errorf("CSV line %v has bad value: %v", c, line[1])
		}

		if m.Values[c].PMC != pmc {
			return fmt.Errorf("Expression output item %v PMC mismatch, expected %v, got %v", c, line[0], m.Values[c].PMC)
		}

		if math.Abs(m.Values[c].Value-value) > 0.000001 {
			return fmt.Errorf("Expression output item %v value mismatch, expected %v, got %v", c, line[1], m.Values[c].Value)
		}
	}

	return nil
}
