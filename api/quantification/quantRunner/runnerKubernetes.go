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
	"sync"
	"time"

	"github.com/pixlise/core/v4/api/config"
	"github.com/pixlise/core/v4/core/kubernetes"
	"github.com/pixlise/core/v4/core/logger"
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

	// Set the namespace based on the type of command being run
	kubeNamespace := cfg.QuantNamespace
	if params.Command != "map" {
		kubeNamespace = cfg.HotQuantNamespace
	}

	// Create channels on which the job can report status and fatal errors
	r.fatalErrors = make(chan error)
	status := make(chan string)
	// Dispatch the piquant run as a Kubernetes Job
	go r.runQuantJob(params, jobId, kubeNamespace, piquantDockerImage, requestorUserId, len(pmcListNames), status)

	// TODO: Quant Notifications
	/*
		if params.Command == "map" {
			err = startQuantNotification(params, notificationStack, creator)
			if err != nil {
				log.Errorf("Failed to send quantification started notification: %v", err)
				err = nil
			}
		}
	*/

	// Wait for all piquant instances to finish
	log.Infof("Waiting for %v pods...", len(pmcListNames))

	for {
		select {
		// Receive status messages from the running Job, exiting once the channel is closed
		case statusMsg, more := <-status:
			r.kubeHelper.Log.Infof("Job %v/%v: %s", kubeNamespace, jobId, statusMsg)
			if !more {
				r.kubeHelper.Log.Infof("Kubernetes pods reported complete")
				return nil
			}
		// Receive fatal errors from the running Job; exiting if any are received
		case kerr := <-r.fatalErrors:
			log.Errorf("Kubernetes Error: %v", kerr.Error())
			return kerr
		}
	}
}

func getPodObject(paramsStr string, params PiquantParams, dockerImage string, jobid, namespace string, requestorUserId string, length int) *apiv1.Pod {
	sec := apiv1.LocalObjectReference{Name: "api-auth"}
	application := "piquant-runner"
	parts := strings.Split(params.PMCListName, ".")
	node := parts[0]
	name := fmt.Sprintf("piquant-%s", params.Command)
	instance := fmt.Sprintf("%s-%s", name, node)
	// Set the serviceaccount for the piquant pods based on namespace
	// Piquant Fit commands will run in the same namespace and share a service account
	// Piquant Map commands (jobs) will run in the piquant-map namespace with a more limited service account
	san := "pixlise-api"
	cpu := "250m"
	if params.Command == "map" {
		san = "piquant-map"
		// PiQuant Map Commands will need much more CPU (and can safely request it since they are running on Fargate nodes)
		cpu = "3500m"
	}

	// Kubernetes doesn't like | in owner name, so we swap it for a _ here
	safeUserId := strings.ReplaceAll(requestorUserId, "|", "_")

	return &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobid + "-" + parts[0],
			Namespace: namespace,
			Labels: map[string]string{
				"pixlise.org/application":     application,
				"pixlise.org/environment":     params.RunTimeEnv,
				"app.kubernetes.io/name":      name,
				"app.kubernetes.io/instance":  instance,
				"app.kubernetes.io/component": application,
				"piquant/command":             params.Command,
				"app":                         node,
				"owner":                       safeUserId,
				"jobid":                       jobid,
				"numberofpods":                strconv.Itoa(length),
			},
		},
		Spec: apiv1.PodSpec{
			ImagePullSecrets:   []apiv1.LocalObjectReference{sec},
			RestartPolicy:      apiv1.RestartPolicyNever,
			ServiceAccountName: san,
			Containers: []apiv1.Container{
				{
					Name:            parts[0],
					Image:           dockerImage,
					ImagePullPolicy: apiv1.PullAlways,
					Resources: apiv1.ResourceRequirements{
						Requests: apiv1.ResourceList{
							// The request determines how much cpu is reserved on the Node and will affect scheduling
							"cpu": resource.MustParse(cpu),
						},
						Limits: apiv1.ResourceList{
							// Allow the pod to use up to 3500m cpu if it's available on the node
							"cpu": resource.MustParse("3500m"),
						},
					},

					Env: []apiv1.EnvVar{
						{Name: "QUANT_PARAMS", Value: paramsStr},
						{Name: "PYTHONUNBUFFERED", Value: "TRUE"},
					},
				},
			},
		},
	}
}

// getJobObject generates a Kubernetes Job Manifest for running piquant.
// It takes in the following parameters:
// - params: a PiquantParams struct containing all parameters needed by piquant
// - paramsStr: json string of the above parameters
// - dockerImage: a string specifying the Docker image to use for the job
// - jobId: a string specifying the unique identifier for the job
// - namespace: a string specifying the namespace in which the job should be created
// - requestorUserId: a string specifying the user ID of the requestor
// - numPods: an integer specifying the number of pods to create for the job
func getJobObject(params PiquantParams, paramsStr, dockerImage, jobId, namespace, requestorUserId string, numPods int) *batchv1.Job {
	imagePullSecret := apiv1.LocalObjectReference{Name: "api-auth"}
	application := "piquant-runner"
	name := fmt.Sprintf("piquant-%s", params.Command)
	// Seconds after job finishes before it is deleted
	jobTtl := 60

	// Set the serviceaccount for the piquant pods based on namespace
	// Piquant Fit commands will run in the same namespace and share a service account
	// Piquant Map commands (jobs) will run in the piquant-map namespace with a more limited service account
	svcAcctName := "pixlise-api"
	cpu := "250m"
	if params.Command == "map" {
		svcAcctName = "piquant-map"
		// PiQuant Map Commands will need much more CPU (and can safely request it since they are running on Fargate nodes)
		cpu = "3500m"
	}

	// Kubernetes doesn't like | in owner name, so we swap it for a _ here
	safeUserId := strings.ReplaceAll(requestorUserId, "|", "_")

	// Pointer management for kubernetes API
	nPods := int32(numPods)
	cm := batchv1.IndexedCompletion
	ttl := int32(jobTtl)

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
			TTLSecondsAfterFinished: &ttl,
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
									"cpu": resource.MustParse(cpu),
								},
								Limits: apiv1.ResourceList{
									// Allow the pod to use up to 3500m cpu if it's available on the node
									"cpu": resource.MustParse("3500m"),
								},
							},

							Env: []apiv1.EnvVar{
								{Name: "QUANT_PARAMS", Value: paramsStr},
								{Name: "PYTHONUNBUFFERED", Value: "TRUE"},
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
	if err != nil {
		return jobStatus, err
	}
	jobStatus = job.Status
	return
}

func (r *kubernetesRunner) runQuantJob(params PiquantParams, jobId, namespace, dockerImage, requestorUserId string, count int, status chan string) {
	defer close(status)
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		r.kubeHelper.Log.Errorf("Failed to serialise JSON params for node: %v", jobId)
		return
	}
	paramsStr := string(paramsJSON)
	jobSpec := getJobObject(params, paramsStr, dockerImage, jobId, namespace, requestorUserId, count)
	job, err := r.kubeHelper.Clientset.BatchV1().Jobs(jobSpec.Namespace).Create(context.Background(), jobSpec, metav1.CreateOptions{})
	if err != nil {
		r.kubeHelper.Log.Errorf("Job create failed for: %v. namespace: %v, count: %v", jobId, namespace, count)
		r.fatalErrors <- err
		return
	}
	// Query the status of the job and report on the number of completed pods
	completed := false
	for !completed {
		time.Sleep(5 * time.Second)
		jobStatus, err := r.getJobStatus(job.Namespace, job.Name)
		if err != nil {
			r.kubeHelper.Log.Errorf("Failed to get job status for: %v. namespace: %v, count: %v", jobId, namespace, count)
			r.fatalErrors <- err
			return
		}
		statusMsg := fmt.Sprintf("%v/%v", jobStatus.Succeeded, count)
		status <- statusMsg
		if jobStatus.Succeeded == int32(count) {
			completed = true
		}
	}
}

func (r *kubernetesRunner) runQuantPod(wg *sync.WaitGroup, params PiquantParams, jobid string, namespace string, dockerImage string, requestorUserId string, count int) {
	defer wg.Done()

	// Make a JSON string out of params so it can be passed in
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		r.kubeHelper.Log.Errorf("Failed to serialise JSON params for node: %v", params.PMCListName)
		return
	}
	paramsStr := string(paramsJSON)

	//log.Debugf("getPodObject for: %v. namespace: %v, count: %v", params.PMCListName, namespace, count)
	pod := getPodObject(paramsStr, params, dockerImage, jobid, namespace, requestorUserId, count)

	co := metav1.CreateOptions{}
	pod, err = r.kubeHelper.Clientset.CoreV1().Pods(pod.Namespace).Create(context.TODO(), pod, co)
	if err != nil {
		r.kubeHelper.Log.Errorf("Pod create failed for: %v. namespace: %v, count: %v", params.PMCListName, namespace, count)
		r.fatalErrors <- err
		return
	}

	// Create Deployment
	r.kubeHelper.Log.Infof("Creating pod for %v in namespace %v...", params.PMCListName, namespace)

	// Now wait for it to finish
	startUnix := time.Now().Unix()
	maxEndUnix := startUnix + config.KubernetesMaxTimeoutSec

	lastPhase := ""

	for currUnix := time.Now().Unix(); currUnix < maxEndUnix; currUnix = time.Now().Unix() {
		// Check kubernetes pod status
		pod, _ := r.kubeHelper.Clientset.CoreV1().Pods(pod.Namespace).Get(context.TODO(), pod.Name, metav1.GetOptions{})

		// TODO: is this needed, now that we log?
		//fmt.Println(pod.Status.Phase)
		//log.Infof("%v phase: %v, pod name: %v, namespace: %v", params.PMCListName, pod.Status.Phase, pod.Name, pod.Namespace)

		phase := string(pod.Status.Phase)
		if lastPhase != phase {
			r.kubeHelper.Log.Infof("%v phase: %v, pod name: %v, namespace: %v", params.PMCListName, pod.Status.Phase, pod.Name, pod.Namespace)
			lastPhase = phase
		}

		if pod.Status.Phase != apiv1.PodRunning && pod.Status.Phase != apiv1.PodPending {
			r.kubeHelper.Log.Infof("Deleting pod: %v from namespace: %v", pod.Name, pod.Namespace)

			deletePolicy := metav1.DeletePropagationForeground
			do := &metav1.DeleteOptions{
				PropagationPolicy: &deletePolicy,
			}
			err := r.kubeHelper.Clientset.CoreV1().Pods(pod.Namespace).Delete(context.TODO(), pod.Name, *do)
			if err != nil {
				r.kubeHelper.Log.Errorf("Failed to remove pod: %v, namespace: %v\n", pod.Name, pod.Namespace)
			}
			break
		}

		time.Sleep(5 * time.Second)
	}
}
