package endpoints

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/v2/api/services"
	"github.com/pixlise/core/v2/core/api"
	"github.com/pixlise/core/v2/core/awsutil"
	"github.com/pixlise/core/v2/core/logger"
)

func Example_testLoggingDebug() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("Activity/2022-11-11/id-123.json"), Body: bytes.NewReader([]byte(`{
    "Instance": "",
    "Time": "2022-11-11T04:56:19Z",
    "Component": "/foo",
    "Message": "the bodyyy",
    "Response": "{\"alive\": true}",
    "Version": "",
    "Params": {
        "method": "GET"
    },
    "Environment": "unit-test",
    "User": "myuserid"
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	var idGen MockIDGenerator
	idGen.ids = []string{"id-123"}

	s := MakeMockSvcs(&mockS3, &idGen, nil, nil)
	s.TimeStamper = &services.MockTimeNowStamper{
		QueuedTimeStamps: []int64{1668142579},
	}

	// Add requestor as a tracked user, so we should see activity saved
	s.Notifications.SetTrack("myuserid", true)

	mockvalidator := api.MockJWTValidator{}
	l := LoggerMiddleware{
		APIServices:  &s,
		JwtValidator: &mockvalidator,
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")

		// In the future we could report back on the status of our DB, or our cache
		// (e.g. Redis) by performing a simple PING, and include them in the response.
		io.WriteString(w, `{"alive": true}`)
	}

	req := httptest.NewRequest("GET", "http://example.com/foo", bytes.NewReader([]byte("the bodyyy")))
	w := httptest.NewRecorder()
	handler(w, req)

	fmt.Printf("%d - %s", w.Code, w.Body.String())

	h := http.HandlerFunc(handler)
	handlerToTest := l.Middleware(h)

	handlerToTest.ServeHTTP(httptest.NewRecorder(), req)

	// Wait a bit for any threads to finish
	time.Sleep(2 * time.Second)

	// Output:
	// 200 - {"alive": true}
}

func Example_testLoggingInfo() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("Activity/2022-11-11/id-123.json"), Body: bytes.NewReader([]byte(`{
    "Instance": "",
    "Time": "2022-11-11T04:56:19Z",
    "Component": "/foo",
    "Message": "the bodyyy",
    "Response": "{\"alive\": true}",
    "Version": "",
    "Params": {
        "method": "GET"
    },
    "Environment": "unit-test",
    "User": "myuserid"
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	var idGen MockIDGenerator
	idGen.ids = []string{"id-123"}

	var ll = logger.LogInfo
	s := MakeMockSvcs(&mockS3, &idGen, nil, &ll)
	s.TimeStamper = &services.MockTimeNowStamper{
		QueuedTimeStamps: []int64{1668142579},
	}

	// Add requestor as a tracked user, so we should see activity saved
	s.Notifications.SetTrack("myuserid", true)

	mockvalidator := api.MockJWTValidator{}
	l := LoggerMiddleware{
		APIServices:  &s,
		JwtValidator: &mockvalidator,
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")

		// In the future we could report back on the status of our DB, or our cache
		// (e.g. Redis) by performing a simple PING, and include them in the response.
		io.WriteString(w, `{"alive": true}`)
	}

	req := httptest.NewRequest("GET", "http://example.com/foo", bytes.NewReader([]byte("the bodyyy")))
	w := httptest.NewRecorder()
	handler(w, req)

	fmt.Printf("%d - %s", w.Code, w.Body.String())

	h := http.HandlerFunc(handler)
	handlerToTest := l.Middleware(h)

	handlerToTest.ServeHTTP(httptest.NewRecorder(), req)

	// Wait a bit for any threads to finish
	time.Sleep(2 * time.Second)

	// Output:
	// 200 - {"alive": true}
}
