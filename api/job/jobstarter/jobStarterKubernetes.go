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

package jobstarter

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pixlise/core/v4/api/config"
	jobrunner "github.com/pixlise/core/v4/api/job/runner"
	"github.com/pixlise/core/v4/core/kubernetes"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/utils"
	batchv1 "k8s.io/api/batch/v1"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

///////////////////////////////////////////////////////////////////////////////////////////
// Runs job in Kubernetes

type kubernetesJobStarter struct {
	fatalErrors chan error
	kubeHelper  kubernetes.KubeHelper
}

// StartJob executes the job in a Kubernetes cluster, creating and monitoring a Kubernetes
// Job resource as the parallel job node workers progress
func (r *kubernetesJobStarter) StartJob(jobDockerImage string, jobConfig JobGroupConfig, apiCfg config.APIConfig, requestorUserId string, log logger.ILogger) error {
	jobId := fmt.Sprintf("job-%v", jobConfig.JobGroupId)

	// Make sure that the kubernetes client is set up
	r.kubeHelper.Kubeconfig = apiCfg.KubeConfig
	r.kubeHelper.Bootstrap(apiCfg.KubernetesLocation, log)

	// We run differently for map commands vs everything else
	kubeNamespace := apiCfg.QuantNamespace // Something that allows many nodes, these will run in the piquant-map namespace with a more limited service account
	svcAcctName := "piquant-map"
	cpu := "3500m"

	// If we're requesting fast-start, we will start on an existing "hot" node so we assume it's a quick, we're less CPU intensive task. We can also probably
	// run in another kubernetes namespace. We have however had issues where this other namespace was failing, so if they're configured the same, do nothing!
	if jobConfig.FastStart {
		cpu = "250m"

		if apiCfg.HotQuantNamespace != apiCfg.QuantNamespace {
			svcAcctName = "pixlise-api"
			kubeNamespace = apiCfg.HotQuantNamespace // Something that does fast startup, these will run in the same namespace and share a service account
		}
	}

	// Create channels on which the job can report status and fatal errors
	r.fatalErrors = make(chan error)
	status := make(chan string)

	// Dispatch as a Kubernetes Job
	go r.runJob(jobConfig, jobId, kubeNamespace, svcAcctName, jobDockerImage, requestorUserId, cpu, apiCfg.EnvironmentName, jobConfig.NodeCount, status, apiCfg.QuantNodeMaxRuntimeSec)

	// Wait for all instances to finish
	log.Infof("Waiting for %v pods...", jobConfig.NodeCount)

	for {
		select {
		// Receive status messages from the running Job, exiting once the channel is closed
		case statusMsg, more := <-status:
			r.kubeHelper.Log.Infof("Job %v/%v: \"%s\"", kubeNamespace, jobId, statusMsg)
			if !more {
				r.kubeHelper.Log.Infof("Kubernetes pods reported complete")
				return nil
			}
		// Receive fatal errors from the running Job; exiting if any are received
		case kerr := <-r.fatalErrors:
			if kerr != nil {
				log.Errorf("Job Error: %v", kerr.Error())
			} else {
				log.Errorf("Unknown Job Error")
			}
			return kerr
		}
	}
}

// makeJobObject generates a Kubernetes Job Manifest for running a job.
// It takes in the following parameters:
// - config: a JobConfig struct containing all parameters needed by job
// - configStr: json string of the above parameters
// - dockerImage: a string specifying the Docker image to use for the job
// - jobId: a string specifying the unique identifier for the job
// - namespace: a string specifying the namespace in which the job should be created
// - requestorUserId: a string specifying the user ID of the requestor
// - numPods: an integer specifying the number of pods to create for the job
func makeJobObject(config jobrunner.JobConfig, configStr, dockerImage, jobId, namespace, svcAcctName, requestorUserId, cpuResource, runtimeEnv string, numPods int, jobTTLSec int64) *batchv1.Job {
	imagePullSecret := apiv1.LocalObjectReference{Name: "api-auth"}
	application := "job-runner" // Used to be piquant-runner
	name := config.JobId        // Used to be piquant-map or piquant-fit (maybe piquant-quant?)

	// Seconds after job finishes before it is deleted
	postJobTTLSec := int32(5 * 60)

	// Kubernetes doesn't like | in owner name, so we swap it for a _ here
	safeUserId := strings.ReplaceAll(requestorUserId, "|", "_")

	// Pointer management for kubernetes API
	nPods := int32(numPods)
	cm := batchv1.IndexedCompletion

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobId,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/component":  application,
				"app.kubernetes.io/name":       name,
				"app.kubernetes.io/instance":   jobId,
				"app.kubernetes.io/managed-by": "pixlise",
				"pixlise.org/application":      application,
				"pixlise.org/environment":      runtimeEnv,
				// Needed? "pixlise.org/piquant-command":  params.Command,
				"pixlise.org/owner":        safeUserId,
				"pixlise.org/jobid":        jobId,
				"pixlise.org/numberofpods": strconv.Itoa(numPods),
			},
		},
		Spec: batchv1.JobSpec{
			Completions:             &nPods,
			Parallelism:             &nPods,
			CompletionMode:          &cm,
			TTLSecondsAfterFinished: &postJobTTLSec,
			ActiveDeadlineSeconds:   &jobTTLSec,
			Template: apiv1.PodTemplateSpec{
				Spec: apiv1.PodSpec{
					ImagePullSecrets:   []apiv1.LocalObjectReference{imagePullSecret},
					RestartPolicy:      apiv1.RestartPolicyNever,
					ServiceAccountName: svcAcctName,
					Containers: []apiv1.Container{
						{
							Name:            application,
							Image:           dockerImage,
							ImagePullPolicy: apiv1.PullAlways,
							Resources: apiv1.ResourceRequirements{
								Requests: apiv1.ResourceList{
									// The request determines how much cpu is reserved on the Node and will affect scheduling
									"cpu": resource.MustParse(cpuResource),
								},
								Limits: apiv1.ResourceList{
									// Allow the pod to use up to 3500m cpu if it's available on the node
									"cpu": resource.MustParse("3500m"),
								},
							},

							Env: []apiv1.EnvVar{
								{Name: jobrunner.JobConfigEnvVar, Value: configStr},
								{Name: "PYTHONUNBUFFERED", Value: "TRUE"},
								/*{Name: "NODE_INDEX", ValueFrom: &apiv1.EnvVarSource{
									FieldRef: &apiv1.ObjectFieldSelector{
										FieldPath: "metadata.annotations['batch.kubernetes.io/job-completion-index']",
									},
								}},*/
							},
						},
					},
				},
			},
		},
	}
}

func (r *kubernetesJobStarter) getJobStatus(namespace, jobId string) (jobStatus batchv1.JobStatus, err error) {
	job, err := r.kubeHelper.Clientset.BatchV1().Jobs(namespace).Get(context.Background(), jobId, metav1.GetOptions{})
	return job.Status, err
}

func (r *kubernetesJobStarter) runJob(jobConfig JobGroupConfig, jobId, namespace, svcAcctName, dockerImage, requestorUserId, cpuResource, runtimeEnv string, count int, status chan string, quantNodeMaxRuntimeSec int32) {
	defer close(status)

	// At this point, we're creating a job which will fan out and create multiple nodes (as needed, see count param), so we make sure the job has the same id as
	// the job group id
	jobConfig.NodeConfig.JobId = jobConfig.JobGroupId

	configJSON, err := json.Marshal(jobConfig.NodeConfig)
	if err != nil {
		r.kubeHelper.Log.Errorf("Failed to serialise JSON config for node: %v", jobId)
		return
	}
	configStr := string(configJSON)

	// Max time job can run for
	jobTTLSec := int64(quantNodeMaxRuntimeSec)

	jobSpec := makeJobObject(jobConfig.NodeConfig, configStr, dockerImage, jobId, namespace, svcAcctName, requestorUserId, cpuResource, runtimeEnv, count, jobTTLSec)

	jobSpecJSON := ""
	if jobSpecJSONBytes, err := json.MarshalIndent(jobSpec, "", utils.PrettyPrintIndentForJSON); err != nil {
		jobSpecJSON = fmt.Sprintf("%+v (failed to read jobSpec - error: %v)", jobSpec, err)
	} else {
		jobSpecJSON = string(jobSpecJSONBytes)
	}

	r.kubeHelper.Log.Infof("runQuantJob creating job for namespace %v, svc account %v: %v", jobSpec.Namespace, jobSpec.Spec.Template.Spec.ServiceAccountName, jobSpecJSON)

	job, err := r.kubeHelper.Clientset.BatchV1().Jobs(jobSpec.Namespace).Create(context.Background(), jobSpec, metav1.CreateOptions{})
	if err != nil {
		err2 := fmt.Errorf("Job create failed for: %v. namespace: %v, count: %v. Error: %v", jobId, namespace, count, err)
		r.kubeHelper.Log.Errorf("%v", err2)
		r.fatalErrors <- err2
		return
	}

	// Query the status of the job and report on the number of completed pods
	startTS := time.Now().Unix()

	lastStatusMsg := ""

	for {
		time.Sleep(5 * time.Second)

		jobStatus, err := r.getJobStatus(job.Namespace, job.Name)
		if err != nil {
			err2 := fmt.Errorf("Failed to get job status for: %v. namespace: %v, count: %v. Error: %v", jobId, namespace, count, err)
			r.kubeHelper.Log.Errorf("%v", err2)
			r.fatalErrors <- err2
			return
		}

		ready := int32(0)
		if jobStatus.Ready != nil {
			ready = *jobStatus.Ready
		}
		statusMsg := fmt.Sprintf("Success %v, Fail %v, Active %v, Ready %v of %v", jobStatus.Succeeded, jobStatus.Failed, jobStatus.Active, ready, count)

		// Only send out a status if there's something new
		if lastStatusMsg != statusMsg {
			status <- statusMsg
			lastStatusMsg = statusMsg
		}

		if jobStatus.Succeeded == int32(count) {
			break
		}

		// If we've been whining for too long, stop logging
		if time.Now().Unix()-startTS > (jobTTLSec + 60) {
			err2 := fmt.Errorf("Timed out monitoring job %v/%v, %v failed nodes, %v succeeded nodes, %v active nodes. Marking job as failed.", namespace, jobId, jobStatus.Failed, jobStatus.Succeeded, jobStatus.Active)
			//			status <- statusMsg
			r.kubeHelper.Log.Errorf("%v", err2)
			r.fatalErrors <- err2
			break
		}
	}
}
