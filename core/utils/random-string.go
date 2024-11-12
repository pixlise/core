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

package utils

import (
	crand "crypto/rand"
	"math/big"
	"math/rand"
)

// Random string generation
// https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-go
const RandomStringChars = "abcdefghijklmnopqrstuvwxyz1234567890"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

func RandStringBytesMaskImpr(n int) string {
	b := make([]byte, n)
	// A rand.Int63() generates 63 random bits, enough for letterIdxMax letters!
	for i, cache, remain := n-1, rand.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = rand.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(RandomStringChars) {
			b[i] = RandomStringChars[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

func RandPassword(length int) (string, error) {
	const (
		lowerChars   = "abcdefghijklmnopqrstuvwxyz"
		upperChars   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		numberChars  = "0123456789"
		specialChars = "!@#$%^&*()-_=+[]{}<>?"
		allChars     = lowerChars + upperChars + numberChars + specialChars
	)

	password := make([]byte, length)

	requiredChars := []byte{
		upperChars[randInt(len(upperChars))],
		numberChars[randInt(len(numberChars))],
		specialChars[randInt(len(specialChars))],
	}

	copy(password, requiredChars)
	for i := len(requiredChars); i < length; i++ {
		password[i] = allChars[randInt(len(allChars))]
	}

	shuffle(password)
	return string(password), nil
}

func randInt(max int) int {
	n, err := crand.Int(crand.Reader, big.NewInt(int64(max)))
	if err != nil {
		panic(err)
	}
	return int(n.Int64())
}

func shuffle(password []byte) {
	for i := range password {
		j := randInt(len(password))
		password[i], password[j] = password[j], password[i]
	}
}
