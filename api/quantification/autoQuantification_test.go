package quantification

import (
	"context"
	"fmt"
	"strings"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/wstestlib"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func Example_getExistingAutoQuants() {
	db := wstestlib.GetDB()

	// Ensure none
	ctx := context.TODO()
	coll := db.Collection(dbCollections.QuantificationsName)
	fmt.Printf("Drop: %v\n", coll.Drop(ctx))

	names := []string{"AutoQuant-PDS(AB)", "AutoQuant-PIXL(AB)", "AutoQuant-PDS(Combined)", "AutoQuant-PIXL(Combined)"}
	existing, err := getExistingAutoQuants("123", names, db)
	fmt.Println("Test missing")
	fmt.Printf("%v\n", err)
	fmt.Printf("Read:%v\n\n", strings.Join(existing, ","))

	autoQuant := &protos.QuantificationSummary{
		Id:     "PIXLAB123",
		ScanId: "123",
		Params: &protos.QuantStartingParameters{
			UserParams: &protos.QuantCreateParams{
				Command:        "map",
				Name:           "AutoQuant-PIXL(AB)",
				ScanId:         "123",
				Elements:       []string{"Na", "Mg"},
				DetectorConfig: "PIXL/PiquantConfigs/v7",
				Parameters:     "-Fe,1",
				QuantMode:      "Combined",
			},
			PmcCount:     51,
			ScanFilePath: "Datasets/104202753/dataset.bin",
		},
		Elements: []string{"Na2O", "MgO"},
		Status: &protos.JobStatus{
			JobId:          "PIXLAB123",
			Status:         5,
			Message:        "Nodes ran: 1",
			EndUnixTimeSec: 1670988052,
			OutputFilePath: "Quantifications/104202753/auth0|62eda29040fd995f305e2322",
			OtherLogFiles:  []string{"node00001_piquant.log", "node00001_stdout.log"},
		},
	}

	_, err = coll.InsertOne(ctx, autoQuant, options.InsertOne())
	fmt.Printf("Insert: %v\n", err)

	//names := []string{"AutoQuant-PIXL(AB)"}
	existing, err = getExistingAutoQuants("123", names, db)
	fmt.Println("Test exists")
	fmt.Printf("%v\n", err)
	fmt.Printf("%v\n", strings.Join(existing, ","))

	// Output:
	// Drop: <nil>
	// Test missing
	// <nil>
	// Read:
	//
	// Insert: <nil>
	// Test exists
	// <nil>
	// AutoQuant-PIXL(AB) (id: PIXLAB123)
}

func Example_readQuantifiablePMCs() {
	expr, err := readDatasetFile("./testdata/LagunaSalinasdataset.bin")
	fmt.Printf("Read Laguna: %v\n", err)
	pmcs, err := readQuantifiablePMCs(expr, "123", &logger.StdOutLoggerForTest{})
	fmt.Printf("PMCRead: %v\n", err)
	fmt.Printf("PMCs: %v\n", pmcs)

	expr, err = readDatasetFile("./testdata/Naltsosdataset.bin")
	fmt.Printf("Read Naltsos: %v\n", err)
	pmcs, err = readQuantifiablePMCs(expr, "123", &logger.StdOutLoggerForTest{})
	fmt.Printf("PMCRead: %v\n", err)
	fmt.Printf("PMCs: %v\n", pmcs)

	// Output:
	// Read Laguna: <nil>
	// PMCRead: <nil>
	// PMCs: []
	// Read Naltsos: <nil>
	// PMCRead: <nil>
	// PMCs: [93 94 95 96 97 98 99 100 101 102 103 104 105 106 107 108 109 110 111 112 113 114 115 116 117 118 119 120 121 122 123 124 125 126 127 128 129 130 131 132 134 135 136 137 138 139 140 141 142 143 144 145 146 147 148 149 150 151 152 153 154 155 156 157 158 159 160 161 162 163 164 165 166 167 168 169 170 171 172 173 175 176 177 178 179 180 181 182 183 184 185 186 187 188 189 190 191 192 193 194 195 196 197 198 199 200 201 202 203 204 205 206 207 208 209 210 211 212 213 214 216]
}
