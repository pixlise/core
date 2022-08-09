// Licensed to NASA JPL under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. NASA JPL licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package awsutil

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/pixlise/core/core/utils"
)

type MockSigner struct {
	Urls []string
}

func (m *MockSigner) GetSignedURL(s3iface.S3API, string, string, time.Duration) (string, error) {
	if len(m.Urls) > 0 {
		url := m.Urls[0]
		m.Urls = m.Urls[1:]
		return url, nil
	}
	return "NO_SIGNED_URL_DEFINED", errors.New("NO_SIGNED_URL_DEFINED")
}

// MockS3Client - mock S3 client for unit tests. Don't forget to call FinishTest() at the end of your test to check
// that all calls to S3 were made, and there were no unexpected calls!
type MockS3Client struct {
	mutex sync.Mutex

	s3iface.S3API

	// Expected requests
	ExpListObjectsV2Input []s3.ListObjectsV2Input
	ExpGetObjectInput     []s3.GetObjectInput
	ExpPutObjectInput     []s3.PutObjectInput
	ExpDeleteObjectInput  []s3.DeleteObjectInput
	ExpCopyObjectInput    []s3.CopyObjectInput

	// Responses replayed as each request comes in
	QueuedListObjectsV2Output []*s3.ListObjectsV2Output
	QueuedGetObjectOutput     []*s3.GetObjectOutput
	QueuedPutObjectOutput     []*s3.PutObjectOutput
	QueuedDeleteObjectOutput  []*s3.DeleteObjectOutput
	QueuedCopyObjectOutput    []*s3.CopyObjectOutput

	AllowGetInAnyOrder    bool
	AllowDeleteInAnyOrder bool

	SkipPutCheckNames []string
}

// NOTE: This function MUST be called at the end of a unit test/example test. Use defer when declaring MockS3Client!
func (m *MockS3Client) FinishTest() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	err := m.getFinishTestResult()

	// If we found something unexpected, print an error so any example tests get this in their input
	// Unit tests which aren't example based will still get our return value
	if err != nil {
		fmt.Println(err)
	}

	return err
}

func (m *MockS3Client) getFinishTestResult() error {
	// Expecting no inputs left
	if len(m.ExpListObjectsV2Input) > 0 {
		return errors.New("Test expected more ListObjectsV2 calls to func")
	}
	if len(m.ExpGetObjectInput) > 0 {
		return errors.New("Test expected more GetObject calls to func")
	}
	if len(m.ExpPutObjectInput) > 0 {
		return errors.New("Test expected more PutObject calls to func")
	}
	if len(m.ExpDeleteObjectInput) > 0 {
		return errors.New("Test expected more DeleteObject calls to func")
	}

	// Expecting nothing left to output
	if len(m.QueuedListObjectsV2Output) > 0 {
		return errors.New("Remaining output ListObjectsV2 for func")
	}
	if len(m.QueuedGetObjectOutput) > 0 {
		return errors.New("Remaining output GetObject for func")
	}
	if len(m.QueuedPutObjectOutput) > 0 {
		return errors.New("Remaining output PutObject for func")
	}
	if len(m.QueuedDeleteObjectOutput) > 0 {
		return errors.New("Remaining output DeleteObject for func")
	}

	return nil
}

/* Go doesn't support generics yet:
https://blog.golang.org/why-generics
So the below can't be done... Or it's a pain in the arse, and we only need to write a few of the interface functions!

func doCheck(name string, input {}interface, expected {}interface, responses {}interface) ({}interface, error) {
	if len(*expected) <= 0 {
		return nil, errors.New("no more inputs expected for "+name)
	}

	// Check it matches the top one
	if (*expected)[0] != *input {
		return nil, errors.New("unexpected input "+name)
	}

	// Don't need this any more!
	(*expected) = (*expected)[1:]

	// Return something
	if len(*responses) <= 0 {
		return nil, errors.New("error in "+name)
	}

	result := (*responses)[0]

	// Don't need this any more!
	(*responses) = (*responses)[1:]

	if result == nil {
		return nil, errors.New("Returning error from "+name)
	}

	return result, nil
}
*/

const ErrNoMoreInputsExpected = "No more inputs expected for "
const ErrWrongInput = "Incorrect input in "
const ErrNothingToReturn = "Nothing to return from "
const ErrReturningError = "Returning error from "

func (m *MockS3Client) ListObjectsV2(input *s3.ListObjectsV2Input) (*s3.ListObjectsV2Output, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	name := "ListObjectsV2"
	expList := &m.ExpListObjectsV2Input
	outputs := &m.QueuedListObjectsV2Output

	if len(*expList) <= 0 {
		return nil, errors.New(ErrNoMoreInputsExpected + name)
	}

	expStr := (*expList)[0].String()

	// Don't need this any more!
	(*expList) = (*expList)[1:]

	// Check it matches the top one
	inpStr := input.String()
	if expStr != inpStr {
		return nil, fmt.Errorf("%v expected: \"%v\" S3 recvd: \"%v\"\n", ErrWrongInput+name, expStr, inpStr)
	}

	// Return something
	if len(*outputs) <= 0 {
		return nil, errors.New(ErrNothingToReturn + name)
	}

	result := (*outputs)[0]

	// Don't need this any more!
	(*outputs) = (*outputs)[1:]

	if result == nil {
		return nil, errors.New(ErrReturningError + name)
	}

	return result, nil
}

func (m *MockS3Client) GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	name := "GetObject"
	expList := &m.ExpGetObjectInput
	outputs := &m.QueuedGetObjectOutput

	if len(*expList) <= 0 {
		return nil, errors.New(ErrNoMoreInputsExpected + name)
	}

	expStr := ""
	inpStr := input.String()
	expListIdx := 0

	if m.AllowGetInAnyOrder {
		// Doing some multi-threaded get-ing, so we search through expected results to find something matching
		for c, expItem := range *expList {
			strExpItem := expItem.String()
			if inpStr == strExpItem {
				expListIdx = c

				// Use this as our expected item
				expStr = strExpItem

				// Remove it from expected list
				if c == 0 {
					(*expList) = (*expList)[1:]
				} else {
					(*expList) = append((*expList)[:c], (*expList)[c+1:]...)
				}
				break
			}
		}
	} else {
		// Expecting them to come in in the order defined... Get next one
		expStr = (*expList)[0].String()

		// Don't need this any more!
		(*expList) = (*expList)[1:]
	}

	// Check it matches expected
	if expStr != inpStr {
		return nil, fmt.Errorf("%v expected: \"%v\" S3 recvd: \"%v\"\n", ErrWrongInput+name, expStr, inpStr)
	}

	// Return something
	if len(*outputs) <= 0 {
		return nil, errors.New(ErrNothingToReturn + name)
	}

	result := (*outputs)[expListIdx]

	// Don't need this any more!
	if expListIdx == 0 {
		(*outputs) = (*outputs)[1:]
	} else {
		(*outputs) = append((*outputs)[:expListIdx], (*outputs)[expListIdx+1:]...)
	}

	if result == nil {
		return nil, awserr.New(s3.ErrCodeNoSuchKey, ErrReturningError+name, nil)
		//return nil, errors.New(ErrReturningError + name)
	}

	return result, nil
}

func getAsStr(r io.Reader) string {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return "ERROR GETTING DATA"
	}
	return string(data)
}

func (m *MockS3Client) PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	name := "PutObject"
	expList := &m.ExpPutObjectInput
	outputs := &m.QueuedPutObjectOutput

	if len(*expList) <= 0 {
		return nil, errors.New(ErrNoMoreInputsExpected + name)
	}

	expItem := (*expList)[0]

	// Don't need this any more!
	(*expList) = (*expList)[1:]

	// Check it matches the top one
	if *input.Bucket != *expItem.Bucket {
		return nil, fmt.Errorf("%v %v - bucket\nexpected: \"%v\"\nS3 recvd: \"%v\"\n", ErrWrongInput, name, *input.Bucket, *expItem.Bucket)
	}

	if *input.Key != *expItem.Key {
		return nil, fmt.Errorf("%v %v - key\nexpected: \"%v\"\nS3 recvd: \"%v\"\n", ErrWrongInput, name, *input.Key, *expItem.Key)
	}

	if !utils.StringInSlice(*input.Key, m.SkipPutCheckNames) {
		inpBody := getAsStr(input.Body)
		expBody := getAsStr(expItem.Body)
		if inpBody != expBody {
			inpBodyLines := strings.Split(inpBody, "\n")
			expBodyLines := strings.Split(expBody, "\n")

			loopToIdx := len(inpBodyLines)
			if l := len(expBodyLines); l > loopToIdx {
				loopToIdx = l
			}

			expLine := ""
			inpLine := ""

			c := 0
			for ; c < loopToIdx; c++ {
				if c >= len(inpBodyLines) || c >= len(expBodyLines) || inpBodyLines[c] != expBodyLines[c] {
					if c < len(inpBodyLines) {
						inpLine = inpBodyLines[c]
					}
					if c < len(expBodyLines) {
						expLine = expBodyLines[c]
					}
					break
				}
			}

			return nil, fmt.Errorf("%v %v - body\nline %v\nexpected: \"%v\"\nS3 recvd: \"%v\"\n", ErrWrongInput, name, c+1, expLine, inpLine)
		}
	}
	// Return something
	if len(*outputs) <= 0 {
		return nil, errors.New(ErrNothingToReturn + name)
	}

	result := (*outputs)[0]

	// Don't need this any more!
	(*outputs) = (*outputs)[1:]

	if result == nil {
		return nil, errors.New(ErrReturningError + name)
	}

	return result, nil
}

func (m *MockS3Client) DeleteObject(input *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	name := "DeleteObject"
	expList := &m.ExpDeleteObjectInput
	outputs := &m.QueuedDeleteObjectOutput

	if len(*expList) <= 0 {
		return nil, errors.New(ErrNoMoreInputsExpected + name)
	}

	expStr := ""
	inpStr := input.String()
	expListIdx := 0

	if m.AllowDeleteInAnyOrder {
		// Doing some multi-threaded get-ing, so we search through expected results to find something matching
		for c, expItem := range *expList {
			strExpItem := expItem.String()
			if inpStr == strExpItem {
				expListIdx = c

				// Use this as our expected item
				expStr = strExpItem

				// Remove it from expected list
				if c == 0 {
					(*expList) = (*expList)[1:]
				} else {
					(*expList) = append((*expList)[:c], (*expList)[c+1:]...)
				}
				break
			}
		}
	} else {
		// Expecting them to come in in the order defined... Get next one
		expStr = (*expList)[0].String()

		// Don't need this any more!
		(*expList) = (*expList)[1:]
	}

	// Check it matches the top one
	if expStr != inpStr {
		return nil, fmt.Errorf("%v %v: expected \"%v\" S3 recvd \"%v\"\n", ErrWrongInput, name, expStr, inpStr)
	}

	// Return something
	if len(*outputs) <= 0 {
		return nil, errors.New(ErrNothingToReturn + name)
	}

	result := (*outputs)[expListIdx]

	// Don't need this any more!
	if expListIdx == 0 {
		(*outputs) = (*outputs)[1:]
	} else {
		(*outputs) = append((*outputs)[:expListIdx], (*outputs)[expListIdx+1:]...)
	}

	if result == nil {
		return nil, errors.New(ErrReturningError + name)
	}

	return result, nil
}

func (m *MockS3Client) CopyObject(input *s3.CopyObjectInput) (*s3.CopyObjectOutput, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	name := "CopyObject"
	expList := &m.ExpCopyObjectInput
	outputs := &m.QueuedCopyObjectOutput

	if len(*expList) <= 0 {
		return nil, errors.New(ErrNoMoreInputsExpected + name)
	}

	expStr := (*expList)[0].String()

	// Don't need this any more!
	(*expList) = (*expList)[1:]

	// Check it matches the top one
	inpStr := input.String()
	if expStr != inpStr {
		return nil, fmt.Errorf("%v expected: \"%v\" S3 recvd: \"%v\"\n", ErrWrongInput+name, expStr, inpStr)
	}

	// Return something
	if len(*outputs) <= 0 {
		return nil, errors.New(ErrNothingToReturn + name)
	}

	result := (*outputs)[0]

	// Don't need this any more!
	(*outputs) = (*outputs)[1:]

	if result == nil {
		return nil, errors.New(ErrReturningError + name)
	}

	return result, nil
}

func (m *MockS3Client) SkipPutChecks(path []string) {
	m.SkipPutCheckNames = path
}
