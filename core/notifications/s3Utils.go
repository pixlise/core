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

package notifications

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/athena"
	"gitlab.com/pixlise/pixlise-go-api/core/awsutil"
	"gitlab.com/pixlise/pixlise-go-api/core/fileaccess"
	"gitlab.com/pixlise/pixlise-go-api/core/logger"
)

type s3utils struct {

}
//GetAllS3Users - Get a minimal list of all users
func (s3 *s3utils) GetAllS3Users(envname string, log logger.ILogger) ([]UserStruct, error) {
	table := envname + "_notifications"
	query := fmt.Sprintf(`SELECT userconfig.name, userconfig.email FROM "userdatabase"."%v";`, table)

	return executeAthenaQuery(query, true, log)
}

//GetS3SubscribersByTopic - Provide a topic, get some subscribers
func (s3 *s3utils) GetS3SubscribersByTopic(topic string, envname string, log logger.ILogger) ([]UserStruct, error) {

	table := envname + "_notifications"
	/*query := fmt.Sprintf(`SELECT userid,
	         i.config.method.ui,
	         i.config.method.sms,
	         i.config.method.email,
	         score.name,
	         score.config.method.ui,
	         score.config.method.sms,
	         score.config.method.email,
	         notifications.hints,
	         userconfig.name,
	         userconfig.email,
	         userconfig.cell,
	         userconfig.data_collection
	FROM ("userdatabase"."%v"
	CROSS JOIN UNNEST(notifications.topics) AS t(i)
	CROSS JOIN UNNEST(notifications.topics) as t(score))
	WHERE i.name = '%v' order by userid;`, table, topic)*/
	query := fmt.Sprintf(`SELECT userid, i.config.method.ui, i.config.method.sms, i.config.method.email, i.name, notifications.topics, notifications.hints, userconfig.name, userconfig.email, userconfig.cell, userconfig.data_collection FROM "userdatabase"."%v", UNNEST(notifications.topics) as t(i) WHERE i.name = '%v'`, table, topic)
	//query := fmt.Sprintf(`SELECT userid, i.config.method.ui, i.config.method.sms, i.config.method.email, notifications.topics, notifications.hints, userconfig.name, userconfig.email, userconfig.cell, userconfig.data_collection FROM "userdatabase"."%v", UNNEST(notifications.topics) as t(i) WHERE i.name = '%v';`, table, topic)
	//fmt.Printf("%v", query)
	return executeAthenaQuery(query, false, log)
}

func getS3SubscribersByID(users []string, envname string, log logger.ILogger) ([]UserStruct, error) {

	table := envname + "_notifications"
	appusers := []string{}
	for _, u := range users {
		if strings.HasPrefix(u, "auth0|") {
			appusers = append(appusers, strings.ReplaceAll(u, "auth0|", ""))
		} else {
			appusers = append(appusers, u)
		}
	}
	j := strings.Join(appusers, "','")
	u := fmt.Sprintf("'%v'", j)

	query := fmt.Sprintf(`SELECT userid, i.config.method.ui, i.config.method.sms, i.config.method.email, notifications.topics, notifications.hints, userconfig.name, userconfig.email, userconfig.cell, userconfig.data_collection FROM "userdatabase"."%v", UNNEST(notifications.topics) as t(i) WHERE userid = '%v';`, table, u)
	//fmt.Println("Executing subscriber lookup: "+query)
	return executeAthenaQuery(query, false, log)
}

//GetS3SubscribersByTopicID - Get Subs by topic
func (s3 *s3utils) GetS3SubscribersByTopicID(users []string, topic string, envname string, log logger.ILogger) ([]UserStruct, error) {
	table := envname + "_notifications"
	appusers := []string{}
	for _, u := range users {
		if strings.HasPrefix(u, "auth0|") {
			appusers = append(appusers, strings.ReplaceAll(u, "auth0|", ""))
		} else {
			appusers = append(appusers, u)
		}
	}
	j := strings.Join(appusers, "','")
	u := fmt.Sprintf("'%v'", j)

	query := fmt.Sprintf(`SELECT userid, i.config.method.ui, i.config.method.sms, i.config.method.email, i.name, notifications.topics, notifications.hints, userconfig.name, userconfig.email, userconfig.cell, userconfig.data_collection FROM "userdatabase"."%v", UNNEST(notifications.topics) as t(i) WHERE i.name = '%v' and userid in (%v)`, table, topic, u)
	//fmt.Printf("%v", query)
	//fmt.Println("Executing subscriber lookup: "+query)
	return executeAthenaQuery(query, false, log)
}

// GetS3SubscribersByEmailTopicID - Get Subs by topic but use their email addresses as the key
func (s3 *s3utils) GetS3SubscribersByEmailTopicID(users []string, topic string, envname string, log logger.ILogger) ([]UserStruct, error) {
	table := envname + "_notifications"
	appusers := []string{}
	for _, u := range users {
		if strings.HasPrefix(u, "auth0|") {
			appusers = append(appusers, strings.ReplaceAll(u, "auth0|", ""))
		} else {
			appusers = append(appusers, u)
		}
	}
	j := strings.Join(appusers, "','")
	u := fmt.Sprintf("'%v'", j)

	query := fmt.Sprintf(`SELECT userid, i.config.method.ui, i.config.method.sms, i.config.method.email, i.name, notifications.topics, notifications.hints, userconfig.name, userconfig.email, userconfig.cell, userconfig.data_collection FROM "userdatabase"."%v", UNNEST(notifications.topics) as t(i) WHERE i.name = '%v' and userconfig.email in (%v)`, table, topic, u)
	//fmt.Printf("%v", query)
	//fmt.Println("Executing subscriber lookup: "+query)
	return executeAthenaQuery(query, false, log)
}

func respToUserStruct(op *athena.GetQueryResultsOutput, log logger.ILogger) ([]UserStruct, error) {
	var users []UserStruct
	for i, r := range op.ResultSet.Rows {
		if i > 0 {
			data := r.Data
			if len(data) < 11 {
				log.Debugf("Unsupported result set, skipping")

			} else {
				datacollect := ""
				userid := ""
				name := ""
				email := ""
				cell := ""

				if (*data[10]).VarCharValue != nil {
					datacollect = *(*data[10]).VarCharValue
				}
				if (*data[0]).VarCharValue != nil {
					userid = *(*data[0]).VarCharValue
				}
				if (*data[7]).VarCharValue != nil {
					name = *(*data[7]).VarCharValue
				}
				if (*data[8]).VarCharValue != nil {
					email = *(*data[8]).VarCharValue
				}
				if (*data[9]).VarCharValue != nil {
					cell = *(*data[9]).VarCharValue
				}

				hints := []string{}
				if (*data[6]).VarCharValue != nil {
					v := strings.TrimPrefix(*(*data[6]).VarCharValue, "[")
					v = strings.TrimSuffix(v, "]")
					hints = strings.Split(v, ",")

				}

				u := UserStruct{
					Userid: userid,
					Notifications: Notifications{
						Topics: processTopics(op.ResultSet.Rows, *data[0].VarCharValue),
						Hints:  hints,
					},
					Config: Config{
						Name:           name,
						Email:          email,
						Cell:           cell,
						DataCollection: datacollect,
					},
				}
				users = append(users, u)
			}
		}
	}
	return users, nil
}

func processTopics(rows []*athena.Row, userid string) []Topics {
	tstruct := []Topics{}
	for _, u := range rows {
		if *u.Data[0].VarCharValue == userid {
			ui, err := strconv.ParseBool(*u.Data[1].VarCharValue)
			sms, err := strconv.ParseBool(*u.Data[2].VarCharValue)
			e, err := strconv.ParseBool(*u.Data[3].VarCharValue)
			if err != nil {
				fmt.Printf(err.Error())
			}
			noteMethod := Method{
				UI:    ui,
				Sms:   sms,
				Email: e,
			}
			topicConfig := NotificationConfig{noteMethod}

			topic := Topics{
				Name:   *u.Data[4].VarCharValue,
				Config: topicConfig,
			}

			tstruct = append(tstruct, topic)
		}
	}
	return tstruct
}

func respToMinimalUserStruct(op *athena.GetQueryResultsOutput) ([]UserStruct, error) {
	var users []UserStruct
	for i, r := range op.ResultSet.Rows {
		if i > 0 {
			data := r.Data
			name := ""
			email := ""
			if (*data[0]).VarCharValue != nil {
				name = *(*data[0]).VarCharValue
			}
			if (*data[1]).VarCharValue != nil {
				email = *(*data[1]).VarCharValue
			}

			u := UserStruct{
				Config: Config{
					Name:  name,
					Email: email,
				},
			}
			users = append(users, u)
		}
	}
	return users, nil
}
func executeAthenaQuery(query string, minimal bool, log logger.ILogger) ([]UserStruct, error) {
	sess, err := awsutil.GetSession()
	svc := athena.New(sess, aws.NewConfig().WithRegion("us-east-1"))
	var s athena.StartQueryExecutionInput
	s.SetQueryString(query)

	var q athena.QueryExecutionContext
	q.SetDatabase("userdatabase")
	s.SetQueryExecutionContext(&q)

	var r athena.ResultConfiguration
	r.SetOutputLocation("s3://devstack-persistencepixlisedata4f446ecf-1corom7nbx3uv/queryoutput/")
	s.SetResultConfiguration(&r)

	result, err := svc.StartQueryExecution(&s)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	var qri athena.GetQueryExecutionInput
	qri.SetQueryExecutionId(*result.QueryExecutionId)

	var qrop *athena.GetQueryExecutionOutput
	duration := time.Duration(2) * time.Second // Pause for 2 seconds

	for {
		qrop, err = svc.GetQueryExecution(&qri)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		if *qrop.QueryExecution.Status.State != "RUNNING" && *qrop.QueryExecution.Status.State != "QUEUED" {
			break
		}
		time.Sleep(duration)

	}
	if *qrop.QueryExecution.Status.State == "SUCCEEDED" {

		var ip athena.GetQueryResultsInput
		ip.SetQueryExecutionId(*result.QueryExecutionId)

		op, err := svc.GetQueryResults(&ip)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		if !minimal {
			return respToUserStruct(op, log)
		}

		return respToMinimalUserStruct(op)

	} else {
		fmt.Printf("EXITED WITH STATUS: %v", qrop.QueryExecution.Status.GoString())
		//fmt.Printf("EXITED WITH MESSAGE: %v", qrop.QueryExecution.)
	}

	fmt.Println(*qrop.QueryExecution.Status.State)

	return nil, nil
}

// UpdateS3UserConfigFile - Update a user configfile by userid
func (s3 *s3utils) UpdateS3UserConfigFile(fs fileaccess.FileAccess, databucket string, userid string, data UserStruct) error {
	// REFACTOR: paths -> filepaths
	path := fmt.Sprintf("/UserContent/notifications/%v.json", userid)

	return fs.WriteJSONNoIndent(databucket, path, data)
}

// FetchS3UserObject - Fetch A User Struct having passed a userid
func (s3 *s3utils) FetchS3UserObject(fs fileaccess.FileAccess, databucket string, userid string, createIfNotExist bool, name string, email string) (UserStruct, error) {
	// REFACTOR: paths -> filepaths
	path := fmt.Sprintf("/UserContent/notifications/%v.json", userid)

	userObj := UserStruct{}
	err := fs.ReadJSON(databucket, path, &userObj, false)

	if err != nil {
		if fs.IsNotFoundError(err) && !createIfNotExist {
			return UserStruct{}, err
		}
		n := NotificationStack{}
		return s3.CreateS3UserObject(n.InitUser(name, email, userid), databucket, fs)
	}

	userObj = remapLegacyTopics(userObj)
	return userObj, nil
}

func remapLegacyTopics(userObj UserStruct) UserStruct {
	for i, t := range userObj.Topics {
		if t.Name == "dataset-updated" {
			userObj.Topics[i].Name = "dataset-spectra-updated"
		}
	}
	return userObj
}

// CreateS3UserObject - Create a new user if none is found
func (s3 *s3utils) CreateS3UserObject(user UserStruct, databucket string, fs fileaccess.FileAccess) (UserStruct, error) {
	return user, s3.UpdateS3UserConfigFile(fs, databucket, user.Userid, user)
}
