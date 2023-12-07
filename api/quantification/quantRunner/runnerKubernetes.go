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

	"github.com/pixlise/core/v3/api/config"
	"github.com/pixlise/core/v3/core/kubernetes"
	"github.com/pixlise/core/v3/core/logger"
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

func (r *kubernetesRunner) RunPiquant(piquantDockerImage string, params PiquantParams, pmcListNames []string, cfg config.APIConfig, requestorUserId string, log logger.ILogger) error {
	var err error
	r.kubeHelper.Kubeconfig = cfg.KubeConfig
	// Setup, create namespace
	jobid := fmt.Sprintf("job-%v", params.JobID)
	r.kubeHelper.Bootstrap(cfg.KubernetesLocation, log)

	log.Infof("Starting %v pods...", len(pmcListNames))

	// Start each container in the namespace
	kubeNamespace := cfg.QuantNamespace
	if params.Command != "map" {
		kubeNamespace = cfg.HotQuantNamespace
	}

	var wg sync.WaitGroup
	r.fatalErrors = make(chan error)
	wgDone := make(chan bool)
	for _, name := range pmcListNames {
		wg.Add(1)

		// Set the pmc name so it gets sent to the container
		params.PMCListName = name
		go r.runQuantJob(&wg, params, jobid, kubeNamespace, piquantDockerImage, requestorUserId, len(pmcListNames))
	}

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
	//wg.Wait()
	go func() {
		wg.Wait()
		close(wgDone)
	}()

	log.Infof("Waiting for %v pods...", len(pmcListNames))

	select {
	case <-wgDone:
		log.Infof("Kubernetes pods reported complete")
		break
	case kerr := <-r.fatalErrors:
		log.Errorf("Kubernetes Error: %v", kerr.Error())
		err = kerr
	}

	return err
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
				"owner":                       requestorUserId,
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

func (r *kubernetesRunner) runQuantJob(wg *sync.WaitGroup, params PiquantParams, jobid string, namespace string, dockerImage string, requestorUserId string, count int) {
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
