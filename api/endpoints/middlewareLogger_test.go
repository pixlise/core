package endpoints

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"gitlab.com/pixlise/pixlise-go-api/api/config"
	"gitlab.com/pixlise/pixlise-go-api/api/esutil"
	"gitlab.com/pixlise/pixlise-go-api/core/api"
	"gitlab.com/pixlise/pixlise-go-api/core/awsutil"
	"gitlab.com/pixlise/pixlise-go-api/core/logger"
)

func Example_testLoggingDebug() {
	testServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer testServer.Close()
	//"Component":"http://example.com/foo","Message":"{\"alive\": true}","Version":"","Params":{"method":"GET"},"Environment":"unit-test","User":"myuserid"}
	var ExpIndexObject = []string{
		`{"Instance":"","Time":"0000-00-00T00:00:00-00:00","Component":"http://example.com/foo","Message":"{\"alive\": true}","Version":"","Params":{"method":"GET"},"Environment":"unit-test","User":"myuserid"}`,
	}
	var ExpRespObject = []string{
		`{"_index":"metrics","_type":"trigger","_id":"B0tzT3wBosV6bFs8gJvY","_version":1,"result":"created","_shards":{"total":2,"successful":2,"failed":0},"_seq_no":8468,"_primary_term":1}`,
		`{"_index":"metrics","_type":"trigger","_id":"B0tzT3wBosV6bFs8gJvY","_version":1,"result":"created","_shards":{"total":2,"successful":2,"failed":0},"_seq_no":8468,"_primary_term":1}`,
	}

	var adjtime = "0000-00-00T00:00:00-00:00"
	d := esutil.DummyElasticClient{}
	foo, err := d.DummyElasticSearchClient(testServer.URL, ExpRespObject, ExpIndexObject, ExpRespObject, &adjtime)

	apiConfig := config.APIConfig{EnvironmentName: "Test"}
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	connection, err := esutil.Connect(foo, apiConfig)

	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("/UserContent/notifications/myuserid.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{Body: ioutil.NopCloser(bytes.NewReader([]byte(`{"userid":"myuserid","notifications":{"topics":[],"hints":["point-select-alt","point-select-z-for-zoom","point-select-shift-for-pan","lasso-z-for-zoom","lasso-shift-for-pan","dwell-exists-test-fm-5x5-full","dwell-exists-069927431"],"uinotifications":[]},"userconfig":{"name":"peternemere","email":"peternemere@gmail.com","cell":"","data_collection":"1.0"}}`)))},
	}

	s := MakeMockSvcs(&mockS3, nil, nil, &connection, nil)

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

	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	fmt.Printf("%d - %s", w.Code, w.Body.String())

	h := http.HandlerFunc(handler)
	handlerToTest := l.Middleware(h)

	handlerToTest.ServeHTTP(httptest.NewRecorder(), req)

	// Output:
	// 200 - {"alive": true}&map[]
}

func Example_testLoggingInfo() {
	testServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer testServer.Close()
	//"Component":"http://example.com/foo","Message":"{\"alive\": true}","Version":"","Params":{"method":"GET"},"Environment":"unit-test","User":"myuserid"}
	var ExpIndexObject = []string{
		`{"Instance":"","Time":"0000-00-00T00:00:00-00:00","Component":"http://example.com/foo","Message":"{\"alive\": true}","Version":"","Params":{"method":"GET"},"Environment":"unit-test","User":"myuserid"}`,
	}
	var ExpRespObject = []string{
		`{"_index":"metrics","_type":"trigger","_id":"B0tzT3wBosV6bFs8gJvY","_version":1,"result":"created","_shards":{"total":2,"successful":2,"failed":0},"_seq_no":8468,"_primary_term":1}`,
		`{"_index":"metrics","_type":"trigger","_id":"B0tzT3wBosV6bFs8gJvY","_version":1,"result":"created","_shards":{"total":2,"successful":2,"failed":0},"_seq_no":8468,"_primary_term":1}`,
	}

	var adjtime = "0000-00-00T00:00:00-00:00"
	d := esutil.DummyElasticClient{}
	foo, err := d.DummyElasticSearchClient(testServer.URL, ExpRespObject, ExpIndexObject, ExpRespObject, &adjtime)

	apiConfig := config.APIConfig{EnvironmentName: "Test"}
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	connection, err := esutil.Connect(foo, apiConfig)

	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("/UserContent/notifications/myuserid.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{Body: ioutil.NopCloser(bytes.NewReader([]byte(`{"userid":"myuserid","notifications":{"topics":[],"hints":["point-select-alt","point-select-z-for-zoom","point-select-shift-for-pan","lasso-z-for-zoom","lasso-shift-for-pan","dwell-exists-test-fm-5x5-full","dwell-exists-069927431"],"uinotifications":[]},"userconfig":{"name":"peternemere","email":"peternemere@gmail.com","cell":"","data_collection":"1.0"}}`)))},
	}

	var ll = logger.LogInfo

	s := MakeMockSvcs(&mockS3, nil, nil, &connection, &ll)

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

	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	fmt.Printf("%d - %s", w.Code, w.Body.String())

	h := http.HandlerFunc(handler)
	handlerToTest := l.Middleware(h)

	handlerToTest.ServeHTTP(httptest.NewRecorder(), req)

	// Output:
	// 200 - {"alive": true}&map[]
}
