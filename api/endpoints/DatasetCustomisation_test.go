// Copyright (c) 2018-2022 California Institute of Technology (“Caltech”). U.S.
// Government sponsorship acknowledged.
// All rights reserved.
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
// * Redistributions of source code must retain the above copyright notice, this
//   list of conditions and the following disclaimer.
// * Redistributions in binary form must reproduce the above copyright notice,
//   this list of conditions and the following disclaimer in the documentation
//   and/or other materials provided with the distribution.
// * Neither the name of Caltech nor its operating division, the Jet Propulsion
//   Laboratory, nor the names of its contributors may be used to endorse or
//   promote products derived from this software without specific prior written
//   permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
// AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
// ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE
// LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
// CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
// SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
// INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
// CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
// ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
// POSSIBILITY OF SUCH DAMAGE.

package endpoints

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/core/awsutil"
)

const artifactManualUploadBucket = "Manual-Uploads"

func Example_datasetCustomMetaGet() {
	const customMetaJSON = `{
"title": "Alien Fossil"
}`
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(artifactManualUploadBucket), Key: aws.String("dataset-addons/abc-123/custom-meta.json"),
		},
		{
			Bucket: aws.String(artifactManualUploadBucket), Key: aws.String("dataset-addons/abc-456/custom-meta.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(customMetaJSON))),
		},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	svcs.Config.ManualUploadBucket = artifactManualUploadBucket
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("GET", "/dataset/meta/abc-123", nil) // Should return empty list, datasets.json fails to download
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("GET", "/dataset/meta/abc-456", nil) // Should return all items. NOTE: tests link creation (though no host name specified so won't have a valid link)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 404
	// dataset custom meta not found
	//
	// 200
	// {
	//     "title": "Alien Fossil"
	// }
}

func Example_datasetCustomMetaPut() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(artifactManualUploadBucket), Key: aws.String("dataset-addons/abc-123/custom-meta.json"), Body: bytes.NewReader([]byte(`{
    "title": "Crater Rim"
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	svcs.Config.ManualUploadBucket = artifactManualUploadBucket
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("PUT", "/dataset/meta/abc-123", bytes.NewReader([]byte("{\"title\": \"Crater Rim\"}"))) // Should return empty list, datasets.json fails to download
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 500
	// AWS Session Not Configured.
}

func Example_datasetCustomImagesList_missingtype() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	svcs.Config.ManualUploadBucket = artifactManualUploadBucket
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("GET", "/dataset/images/abc-111", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 404
	// 404 page not found
}

func Example_datasetCustomImagesList_rgbu() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{Bucket: aws.String(artifactManualUploadBucket), Prefix: aws.String("dataset-addons/abc-111/RGBU/")},
		{Bucket: aws.String(artifactManualUploadBucket), Prefix: aws.String("dataset-addons/abc-123/RGBU/")},
	}
	mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{
		nil,
		{
			Contents: []*s3.Object{
				{Key: aws.String("dataset-addons/abc-123/RGBU/nirgbuv.tif")},
				{Key: aws.String("dataset-addons/abc-123/RGBU/shouldnt be here.txt")},
				{Key: aws.String("dataset-addons/abc-123/RGBU/another.tif")},
			},
		},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	svcs.Config.ManualUploadBucket = artifactManualUploadBucket
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("GET", "/dataset/images/abc-111/rgbu", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("GET", "/dataset/images/abc-123/rgbu", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 404
	// custom images not found
	//
	// 200
	// [
	//     "nirgbuv.tif",
	//     "shouldnt be here.txt",
	//     "another.tif"
	// ]
}

func Example_datasetCustomImagesList_unaligned() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{Bucket: aws.String(artifactManualUploadBucket), Prefix: aws.String("dataset-addons/abc-111/UNALIGNED/")},
		{Bucket: aws.String(artifactManualUploadBucket), Prefix: aws.String("dataset-addons/abc-123/UNALIGNED/")},
	}
	mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{
		nil,
		{
			Contents: []*s3.Object{
				{Key: aws.String("dataset-addons/abc-123/UNALIGNED/mastcam-123.jpg")},
				{Key: aws.String("dataset-addons/abc-123/UNALIGNED/shouldnt be here.txt")},
				{Key: aws.String("dataset-addons/abc-123/UNALIGNED/mosaic.png")},
			},
		},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	svcs.Config.ManualUploadBucket = artifactManualUploadBucket
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("GET", "/dataset/images/abc-111/unaligned", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("GET", "/dataset/images/abc-123/unaligned", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 404
	// custom images not found
	//
	// 200
	// [
	//     "mastcam-123.jpg",
	//     "shouldnt be here.txt",
	//     "mosaic.png"
	// ]
}

func Example_datasetCustomImagesList_matched() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{Bucket: aws.String(artifactManualUploadBucket), Prefix: aws.String("dataset-addons/abc-111/MATCHED/")},
		{Bucket: aws.String(artifactManualUploadBucket), Prefix: aws.String("dataset-addons/abc-123/MATCHED/")},
	}
	mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{
		nil,
		{
			Contents: []*s3.Object{
				{Key: aws.String("dataset-addons/abc-123/RGBU/watson-123.jpg")},
				{Key: aws.String("dataset-addons/abc-123/RGBU/watson-123.json")},
				{Key: aws.String("dataset-addons/abc-123/RGBU/shouldnt be here.txt")},
				{Key: aws.String("dataset-addons/abc-123/RGBU/watson-777.png")},
				{Key: aws.String("dataset-addons/abc-123/RGBU/watson-777-meta.json")},
				{Key: aws.String("dataset-addons/abc-123/RGBU/watson-33.png")},
				{Key: aws.String("dataset-addons/abc-123/RGBU/watson-33.json")},
				{Key: aws.String("dataset-addons/abc-123/RGBU/mosaic.png")},
			},
		},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	svcs.Config.ManualUploadBucket = artifactManualUploadBucket
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("GET", "/dataset/images/abc-111/matched", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("GET", "/dataset/images/abc-123/matched", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 404
	// custom images not found
	//
	// 200
	// [
	//     "watson-123.jpg",
	//     "watson-33.png"
	// ]
}

func Example_datasetCustomImagesGet_rgbu() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	svcs.Config.ManualUploadBucket = artifactManualUploadBucket
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("GET", "/dataset/images/abc-111/rgbu/rgbu.tif", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
	// {
	//     "download-link": "https:///dataset/download/abc-111/rgbu.tif?loadCustomType=rgbu"
	// }
}

func Example_datasetCustomImagesGet_unaligned() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	svcs.Config.ManualUploadBucket = artifactManualUploadBucket
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("GET", "/dataset/images/abc-111/unaligned/mastcamZ.png", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
	// {
	//     "download-link": "https:///dataset/download/abc-111/mastcamZ.png?loadCustomType=unaligned"
	// }
}

func Example_datasetCustomImagesGet_matched() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(artifactManualUploadBucket), Key: aws.String("dataset-addons/abc-111/MATCHED/watson-123.json"),
		},
		{
			Bucket: aws.String(artifactManualUploadBucket), Key: aws.String("dataset-addons/abc-123/MATCHED/watson-33.json"),
		},
		{
			Bucket: aws.String(DatasetsBucketForUnitTest), Key: aws.String("Datasets/abc-123/dataset.bin"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
    "aligned-beam-pmc": 77,
    "matched-image": "watson-33.png",
    "x-offset": 11,
    "y-offset": 12,
    "x-scale": 1.4,
    "y-scale": 1.5
}`))),
		},
		nil,
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	svcs.Config.ManualUploadBucket = artifactManualUploadBucket
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("GET", "/dataset/images/abc-111/matched/watson-123.jpg", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("GET", "/dataset/images/abc-123/matched/watson-33.png", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 404
	// dataset custom image meta not found
	//
	// 200
	// {
	//     "alignedImageLink": "",
	//     "download-link": "https:///dataset/download/abc-123/watson-33.png?loadCustomType=matched",
	//     "aligned-beam-pmc": 77,
	//     "matched-image": "watson-33.png",
	//     "x-offset": 11,
	//     "y-offset": 12,
	//     "x-scale": 1.4,
	//     "y-scale": 1.5
	// }
}

func Example_datasetCustomImagesPost_badtype() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	svcs.Config.ManualUploadBucket = artifactManualUploadBucket
	apiRouter := MakeRouter(svcs)

	// Bad type
	req, _ := http.NewRequest("POST", "/dataset/images/abc-111/badtype/nirgbuv.png", bytes.NewBuffer([]byte{84, 73, 70}))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 400
	// Invalid custom image type: "badtype"
}

func Example_datasetCustomImagesPost_badfilename() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	svcs.Config.ManualUploadBucket = artifactManualUploadBucket
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("POST", "/dataset/images/abc-111/rgbu/noextension-file-name", bytes.NewBuffer([]byte{84, 73, 70}))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 400
	// Invalid file name: "noextension-file-name"
}

func Example_datasetCustomImagesPost_rgbu() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	// Expecting uploaded image and JSON file
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(artifactManualUploadBucket), Key: aws.String("dataset-addons/abc-111/RGBU/nirgbuv.tif"), Body: bytes.NewReader([]byte("TIF")),
		},
	}

	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	svcs.Config.ManualUploadBucket = artifactManualUploadBucket
	apiRouter := MakeRouter(svcs)

	// Bad image type
	req, _ := http.NewRequest("POST", "/dataset/images/abc-111/rgbu/nirgbuv.png", bytes.NewBuffer([]byte{84, 73, 70}))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// No body
	req, _ = http.NewRequest("POST", "/dataset/images/abc-111/rgbu/nirgbuv.tif", bytes.NewBuffer([]byte{}))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("POST", "/dataset/images/abc-111/rgbu/nirgbuv.tif", bytes.NewBuffer([]byte{84, 73, 70}))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 400
	// Invalid image file type: "nirgbuv.png"
	//
	// 400
	// No image data sent
	//
	// 200
}

func Example_datasetCustomImagesPost_unaligned() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	// Expecting uploaded image and JSON file
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(artifactManualUploadBucket), Key: aws.String("dataset-addons/abc-111/UNALIGNED/mastcam.png"), Body: bytes.NewReader([]byte("PNG")),
		},
	}

	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	svcs.Config.ManualUploadBucket = artifactManualUploadBucket
	apiRouter := MakeRouter(svcs)

	// Bad image type
	req, _ := http.NewRequest("POST", "/dataset/images/abc-111/unaligned/mastcam.tif", bytes.NewBuffer([]byte{80, 78, 71}))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// No body
	req, _ = http.NewRequest("POST", "/dataset/images/abc-111/unaligned/mastcam.png", bytes.NewBuffer([]byte{}))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("POST", "/dataset/images/abc-111/unaligned/mastcam.png", bytes.NewBuffer([]byte{80, 78, 71}))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 400
	// Invalid image file type: "mastcam.tif"
	//
	// 400
	// No image data sent
	//
	// 200
}

func Example_datasetCustomImagesPost_matched() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	// Expecting uploaded image and JSON file
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(artifactManualUploadBucket), Key: aws.String("dataset-addons/abc-111/MATCHED/watson-444.json"), Body: bytes.NewReader([]byte(`{
    "aligned-beam-pmc": 88,
    "matched-image": "watson-444.png",
    "x-offset": 11,
    "y-offset": 22,
    "x-scale": 1.23,
    "y-scale": 1.1
}`)),
		},
		{
			Bucket: aws.String(artifactManualUploadBucket), Key: aws.String("dataset-addons/abc-111/MATCHED/watson-444.png"), Body: bytes.NewReader([]byte("PNG")),
		},
	}

	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	svcs.Config.ManualUploadBucket = artifactManualUploadBucket
	apiRouter := MakeRouter(svcs)

	// Missing aligned-beam-pmc
	req, _ := http.NewRequest("POST", "/dataset/images/abc-111/matched/watson-444.png?x-scale=1.23&y-scale=1.1&x-offset=11&y-offset=22", bytes.NewBuffer([]byte{80, 78, 71}))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Missing x-scale
	req, _ = http.NewRequest("POST", "/dataset/images/abc-111/matched/watson-444.png?y-scale=1.1&x-offset=11&y-offset=22&aligned-beam-pmc=88", bytes.NewBuffer([]byte{80, 78, 71}))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// x-scale is not float
	req, _ = http.NewRequest("POST", "/dataset/images/abc-111/matched/watson-444.png?x-scale=Large&y-scale=1.1&x-offset=11&y-offset=22&aligned-beam-pmc=88", bytes.NewBuffer([]byte{80, 78, 71}))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Aligned-beam-pmc is empty
	req, _ = http.NewRequest("POST", "/dataset/images/abc-111/matched/watson-444.png?x-scale=1.23&y-scale=1.1&x-offset=11&y-offset=22&aligned-beam-pmc=", bytes.NewBuffer([]byte{80, 78, 71}))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Bad image type
	req, _ = http.NewRequest("POST", "/dataset/images/abc-111/matched/watson-444.gif?x-scale=1.23&y-scale=1.1&x-offset=11&y-offset=22&aligned-beam-pmc=88", bytes.NewBuffer([]byte{80, 78, 71}))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Empty image body
	req, _ = http.NewRequest("POST", "/dataset/images/abc-111/matched/watson-444.png?x-scale=1.23&y-scale=1.1&x-offset=11&y-offset=22&aligned-beam-pmc=88", bytes.NewBuffer([]byte{}))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Works
	req, _ = http.NewRequest("POST", "/dataset/images/abc-111/matched/watson-444.png?x-scale=1.23&y-scale=1.1&x-offset=11&y-offset=22&aligned-beam-pmc=88", bytes.NewBuffer([]byte{80, 78, 71})) // Spells PNG in ascii
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 400
	// Missing query parameter "aligned-beam-pmc" for matched image: "watson-444.png"
	//
	// 400
	// Missing query parameter "x-scale" for matched image: "watson-444.png"
	//
	// 400
	// Query parameter "x-scale" was not a float, for matched image: "watson-444.png"
	//
	// 400
	// Query parameter "aligned-beam-pmc" was not an int, for matched image: "watson-444.png"
	//
	// 400
	// Invalid image file type: "watson-444.gif"
	//
	// 400
	// No image data sent
	//
	// 200
}

func Example_datasetCustomImagesPut() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(artifactManualUploadBucket), Key: aws.String("dataset-addons/abc-111/MATCHED/doesnt-exist.json"),
		},
		{
			Bucket: aws.String(artifactManualUploadBucket), Key: aws.String("dataset-addons/abc-111/MATCHED/watson-444.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
    "aligned-beam-pmc": 77,
    "matched-image": "watson-444.png",
    "x-offset": 11,
    "y-offset": 12,
    "x-scale": 1.4,
    "y-scale": 1.5
}`))),
		},
	}

	// Expecting uploaded JSON file ONCE
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(artifactManualUploadBucket), Key: aws.String("dataset-addons/abc-111/MATCHED/watson-444.json"), Body: bytes.NewReader([]byte(`{
    "aligned-beam-pmc": 88,
    "matched-image": "watson-444.png",
    "x-offset": 12,
    "y-offset": 23,
    "x-scale": 1.23,
    "y-scale": 1.1
}`)),
		},
	}

	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	svcs.Config.ManualUploadBucket = artifactManualUploadBucket
	apiRouter := MakeRouter(svcs)

	// Missing aligned-beam-pmc
	req, _ := http.NewRequest("PUT", "/dataset/images/abc-111/matched/watson-444.png?x-scale=1.23&y-scale=1.1&x-offset=11&y-offset=22", bytes.NewBuffer([]byte{}))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Missing x-scale
	req, _ = http.NewRequest("PUT", "/dataset/images/abc-111/matched/watson-444.png?y-scale=1.1&x-offset=11&y-offset=22&aligned-beam-pmc=88", bytes.NewBuffer([]byte{}))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// x-scale is not float
	req, _ = http.NewRequest("PUT", "/dataset/images/abc-111/matched/watson-444.png?x-scale=Large&y-scale=1.1&x-offset=11&y-offset=22&aligned-beam-pmc=88", bytes.NewBuffer([]byte{}))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Aligned-beam-pmc is empty
	req, _ = http.NewRequest("PUT", "/dataset/images/abc-111/matched/watson-444.png?x-scale=1.23&y-scale=1.1&x-offset=11&y-offset=22&aligned-beam-pmc=", bytes.NewBuffer([]byte{}))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Bad image type
	req, _ = http.NewRequest("PUT", "/dataset/images/abc-111/unaligned/watson-444.png?x-scale=1.23&y-scale=1.1&x-offset=11&y-offset=22&aligned-beam-pmc=88", bytes.NewBuffer([]byte{}))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Bad image name
	req, _ = http.NewRequest("PUT", "/dataset/images/abc-111/matched/doesnt-exist.png?x-scale=1.23&y-scale=1.1&x-offset=11&y-offset=22&aligned-beam-pmc=88", bytes.NewBuffer([]byte{}))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Works
	req, _ = http.NewRequest("PUT", "/dataset/images/abc-111/matched/watson-444.png?x-scale=1.23&y-scale=1.1&x-offset=12&y-offset=23&aligned-beam-pmc=88", bytes.NewBuffer([]byte{}))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 400
	// Missing query parameter "aligned-beam-pmc" for matched image: "watson-444.png"
	//
	// 400
	// Missing query parameter "x-scale" for matched image: "watson-444.png"
	//
	// 400
	// Query parameter "x-scale" was not a float, for matched image: "watson-444.png"
	//
	// 400
	// Query parameter "aligned-beam-pmc" was not an int, for matched image: "watson-444.png"
	//
	// 400
	// Invalid custom image type: "unaligned"
	//
	// 404
	// doesnt-exist.json not found
	//
	// 200
}

func Example_datasetCustomImagesDelete() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpDeleteObjectInput = []s3.DeleteObjectInput{
		{
			Bucket: aws.String(artifactManualUploadBucket), Key: aws.String("dataset-addons/abc-123/UNALIGNED/unaligned-222.jpg"),
		},
		{
			Bucket: aws.String(artifactManualUploadBucket), Key: aws.String("dataset-addons/abc-123/UNALIGNED/unaligned-222.jpg"),
		},
		{
			Bucket: aws.String(artifactManualUploadBucket), Key: aws.String("dataset-addons/abc-123/RGBU/nirgbuv-333.tif"),
		},
		{
			Bucket: aws.String(artifactManualUploadBucket), Key: aws.String("dataset-addons/abc-123/RGBU/nirgbuv-333.tif"),
		},
		{
			Bucket: aws.String(artifactManualUploadBucket), Key: aws.String("dataset-addons/abc-123/MATCHED/watson-222.json"),
		},
		{
			Bucket: aws.String(artifactManualUploadBucket), Key: aws.String("dataset-addons/abc-123/MATCHED/watson-222.json"),
		},
		{
			Bucket: aws.String(artifactManualUploadBucket), Key: aws.String("dataset-addons/abc-123/MATCHED/watson-222.jpg"),
		},
	}

	mockS3.QueuedDeleteObjectOutput = []*s3.DeleteObjectOutput{
		nil, // unaligned missing
		{},
		nil, // rgbu missing
		{},
		nil, // matched JSON missing
		{},  // matched json
		{},  // matched image
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	svcs.Config.ManualUploadBucket = artifactManualUploadBucket
	apiRouter := MakeRouter(svcs)

	// Missing type
	req, _ := http.NewRequest("DELETE", "/dataset/images/abc-123/watson-111.jpg", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Bad type
	req, _ = http.NewRequest("DELETE", "/dataset/images/abc-123/badtype/watson-222.jpg", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Unaligned, fail
	req, _ = http.NewRequest("DELETE", "/dataset/images/abc-123/unaligned/unaligned-222.jpg", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Unaligned, OK
	req, _ = http.NewRequest("DELETE", "/dataset/images/abc-123/unaligned/unaligned-222.jpg", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// RGBU, fail
	req, _ = http.NewRequest("DELETE", "/dataset/images/abc-123/rgbu/nirgbuv-333.tif", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// RGBU, OK
	req, _ = http.NewRequest("DELETE", "/dataset/images/abc-123/rgbu/nirgbuv-333.tif", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Matched, fail
	req, _ = http.NewRequest("DELETE", "/dataset/images/abc-123/matched/watson-222.jpg", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Matched, OK
	req, _ = http.NewRequest("DELETE", "/dataset/images/abc-123/matched/watson-222.jpg", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 405
	//
	// 400
	// Invalid custom image type: "badtype"
	//
	// 404
	// unaligned-222.jpg not found
	//
	// 200
	//
	// 404
	// nirgbuv-333.tif not found
	//
	// 200
	//
	// 404
	// watson-222.json not found
	//
	// 200
	//
}
