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

package quantRunner

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pixlise/core/v4/api/config"
	"github.com/pixlise/core/v4/core/kubernetes"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/utils"
	batchv1 "k8s.io/api/batch/v1"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

///////////////////////////////////////////////////////////////////////////////////////////
// PIQUANT in Kubernetes

type kubernetesRunner struct {
	fatalErrors chan error
	kubeHelper  kubernetes.KubeHelper
}

// RunPiquant executes the piquant command in a Kubernetes cluster, creating and
// monitoring a Kubernetes Job resource as the parallel piquant workers progress
func (r *kubernetesRunner) RunPiquant(piquantDockerImage string, params PiquantParams, pmcListNames []string, cfg config.APIConfig, requestorUserId string, log logger.ILogger) error {
	jobId := fmt.Sprintf("job-%v", params.JobID)

	// Make sure that the kubernetes client is set up
	r.kubeHelper.Kubeconfig = cfg.KubeConfig
	r.kubeHelper.Bootstrap(cfg.KubernetesLocation, log)

	// We run differently for map commands vs everything else
	kubeNamespace := cfg.QuantNamespace // Something that allows many nodes, these will run in the piquant-map namespace with a more limited service account
	svcAcctName := "piquant-map"
	cpu := "3500m"

	// If we're NOT a map command, we're less CPU intensive. We can also probably run in another kubernetes namespace. We have however had issues
	// where this other namespace was failing, so if they're configured the same, do nothing!
	if params.Command != "map" {
		cpu = "250m"

		if cfg.HotQuantNamespace != cfg.QuantNamespace {
			svcAcctName = "pixlise-api"
			kubeNamespace = cfg.HotQuantNamespace // Something that does fast startup, these will run in the same namespace and share a service account
		}
	}

	// Create channels on which the job can report status and fatal errors
	r.fatalErrors = make(chan error)
	status := make(chan string)

	// Dispatch the piquant run as a Kubernetes Job
	go r.runQuantJob(params, jobId, kubeNamespace, svcAcctName, piquantDockerImage, requestorUserId, cpu, len(pmcListNames), status, cfg.QuantNodeMaxRuntimeSec)

	// Wait for all piquant instances to finish
	log.Infof("Waiting for %v pods...", len(pmcListNames))

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
				log.Errorf("Quant Error: %v", kerr.Error())
			} else {
				log.Errorf("Unknown Quant Error")
			}
			return kerr
		}
	}
}

// makeJobObject generates a Kubernetes Job Manifest for running piquant.
// It takes in the following parameters:
// - params: a PiquantParams struct containing all parameters needed by piquant
// - paramsStr: json string of the above parameters
// - dockerImage: a string specifying the Docker image to use for the job
// - jobId: a string specifying the unique identifier for the job
// - namespace: a string specifying the namespace in which the job should be created
// - requestorUserId: a string specifying the user ID of the requestor
// - numPods: an integer specifying the number of pods to create for the job
func makeJobObject(params PiquantParams, paramsStr, dockerImage, jobId, namespace, svcAcctName, requestorUserId, cpuResource string, numPods int, jobTTLSec int64) *batchv1.Job {
	imagePullSecret := apiv1.LocalObjectReference{Name: "api-auth"}
	application := "piquant-runner"
	name := fmt.Sprintf("piquant-%s", params.Command)

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
				"pixlise.org/environment":      params.RunTimeEnv,
				"pixlise.org/piquant-command":  params.Command,
				"pixlise.org/owner":            safeUserId,
				"pixlise.org/jobid":            jobId,
				"pixlise.org/numberofpods":     strconv.Itoa(numPods),
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
								{Name: "QUANT_PARAMS", Value: paramsStr},
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

func (r *kubernetesRunner) getJobStatus(namespace, jobId string) (jobStatus batchv1.JobStatus, err error) {
	job, err := r.kubeHelper.Clientset.BatchV1().Jobs(namespace).Get(context.Background(), jobId, metav1.GetOptions{})
	return job.Status, err
}

func (r *kubernetesRunner) runQuantJob(params PiquantParams, jobId, namespace, svcAcctName, dockerImage, requestorUserId, cpuResource string, count int, status chan string, quantNodeMaxRuntimeSec int32) {
	defer close(status)
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		r.kubeHelper.Log.Errorf("Failed to serialise JSON params for node: %v", jobId)
		return
	}
	paramsStr := string(paramsJSON)

	// Max time job can run for
	jobTTLSec := int64(quantNodeMaxRuntimeSec)

	jobSpec := makeJobObject(params, paramsStr, dockerImage, jobId, namespace, svcAcctName, requestorUserId, cpuResource, count, jobTTLSec)

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
