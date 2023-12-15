package coreg

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	protos "github.com/pixlise/core/v3/generated-protos"
)

func TestNewJobFromMVExport(t *testing.T) {
	type args struct {
		mve *protos.MarsViewerExport
	}

	// Read as JSON string
	testData, err := os.ReadFile("./test-data/Marsviewer_PIXLISE_export_2023-05-24_d1df5b76-d4d9-4756-b3eb-33889f7fae58.json")
	if err != nil {
		t.Fatalf("Failed to read test data file: %s", err)
	}

	// Read into JSON object
	var sampleMve protos.MarsViewerExport
	err = json.Unmarshal(testData, &sampleMve)

	// Serialise to proto binary format
	//proto.Marshal()

	// Read from proto binary format
	//var sampleMve protos.MarsViewerExport
	//err = protojson.Unmarshal(testData, &sampleMve)
	if err != nil {
		t.Fatalf("Failed to unmarshal test data: %s", err)
	}

	job := CoregJobRequest{
		ContextImageUrls: []string{"s3://m20-ids-g-data-g66bt/ods/dev/sol/00614/ids/rdr/shrlc/SC3_0614_0721474493_226RAS_N0301172SRLC11360_0000LMJ03.IMG"},
		MappedImageUrls:  []string{"s3://m20-ids-g-data-g66bt/ods/dev/sol/00614/ids/rdr/shrlc/SIF_0614_0721455441_734RAS_N0301172SRLC00643_0000LMJ03.IMG"},
		WarpedImageUrls:  []string{"s3://m20-dev-ids-crisp-imgcoregi/crisp_data/ICM-SC3_0614_0721480213_394RAS_N0301172SRLC10600_0000-SIF_0614_0721455441_734RAS_N0301172SRLC00643_0000-J01.VIC/09cd570f0963157fbf39575de1da4e73/crisp_data/ods/dev/sol/00614/ids/rdr/shrlc/warped-zoom_1-SN100D0-SIF_0614_0721455441_734RAS_N0301172SRLC00643_0000LMJ03.VIC"},
		BaseImageUrl:     "s3://m20-ids-g-data-g66bt/ods/dev/sol/00614/ids/rdr/shrlc/SC3_0614_0721480213_394RAS_N0301172SRLC10600_0000LMJ03.IMG",
		JobID:            "foobar",
	}
	tests := []struct {
		name string
		args args
		want *CoregJobRequest
	}{
		{"sherlock_sample", args{&sampleMve}, &job},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewJobFromMVExport(tt.want.JobID, tt.args.mve)
			got.JobID = tt.want.JobID
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewJobFromMVExport() = %v, want %v", got, tt.want)
			}
		})
	}
}
