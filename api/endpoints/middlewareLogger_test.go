package endpoints

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/v4/core/awsutil"
	"github.com/pixlise/core/v4/core/idgen"
	"github.com/pixlise/core/v4/core/jwtparser"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/timestamper"
)

func Test_testLoggingDebug(t *testing.T) {
	runMiddlewareLoggingTest(t, nil)
}

func Test_testLoggingInfo(t *testing.T) {
	var ll = logger.LogInfo
	runMiddlewareLoggingTest(t, &ll)
}

func runMiddlewareLoggingTest(t *testing.T, logLevel *logger.LogLevel) {
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

	idGen := idgen.MockIDGenerator{
		IDs: []string{"id-123"},
	}

	s := MakeMockSvcs(&mockS3, &idGen, logLevel)
	s.TimeStamper = &timestamper.MockTimeNowStamper{
		QueuedTimeStamps: []int64{1668142579},
	}

	mockvalidator := jwtparser.MockJWTValidator{}
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

	h := http.HandlerFunc(handler)
	handlerToTest := l.Middleware(h)

	handlerToTest.ServeHTTP(httptest.NewRecorder(), req)

	// Wait a bit for any threads to finish
	time.Sleep(2 * time.Second)

	checkResult(t, w, 200, "{\"alive\": true}")
}
