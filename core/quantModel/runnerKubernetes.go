// Copyright (c) 2018-2022 California Institute of Technology (“Caltech”). U.S.
// Government sponsorship acknowledged.
// All rights reserved.
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
// * Redistributions of source code must retain the above copyright notice, this
//   list of conditions and the following disclaimer.
// * Redistributions in binary form must reproduce the above copyright notice,
//   this list of conditions and the following disclaimer in the documentation
//   and/or other materials provided with the distribution.
// * Neither the name of Caltech nor its operating division, the Jet Propulsion
//   Laboratory, nor the names of its contributors may be used to endorse or
//   promote products derived from this software without specific prior written
//   permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
// AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
// ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE
// LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
// CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
// SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
// INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
// CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
// ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
// POSSIBILITY OF SUCH DAMAGE.

package quantModel

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pixlise/core/core/kubernetes"
	"github.com/pixlise/core/core/notifications"
	"os"
	"strconv"
	"strings"

	"github.com/pixlise/core/core/logger"

	"github.com/pixlise/core/core/pixlUser"
	"k8s.io/apimachinery/pkg/api/resource"

	"sync"
	"time"

	"github.com/pixlise/core/api/config"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

///////////////////////////////////////////////////////////////////////////////////////////
// PIQUANT in Kubernetes

type kubernetesRunner struct {
}

var fatalErrors chan error
var kubeHelper kubernetes.KubeHelper

func (r kubernetesRunner) runPiquant(piquantDockerImage string, params PiquantParams, pmcListNames []string, cfg config.APIConfig, notificationStack notifications.NotificationManager, creator pixlUser.UserInfo, log logger.ILogger) error {
	log.Infof("kubernetesRunner runPiquant called...")

	kubeHelper.Kubeconfig = cfg.KubeConfig
	// Setup, create namespace
	jobid := fmt.Sprintf("job-%v", params.JobID)
	kubeHelper.Bootstrap(cfg.KubernetesLocation, log)

	log.Infof("Starting %v pods...", len(pmcListNames))

	// initNamespace(params, cfg.DockerLoginString)
	// Start each container in the namespace
	var wg sync.WaitGroup
	fatalErrors = make(chan error)
	wgDone := make(chan bool)
	for _, name := range pmcListNames {
		wg.Add(1)

		// Set the pmc name so it gets sent to the container
		params.PMCListName = name
		go runQuantJob(&wg, params, jobid, cfg.QuantNamespace, piquantDockerImage, creator, len(pmcListNames), log)
	}

	err := startQuantNotification(params, notificationStack, creator)

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
	case kerr := <-fatalErrors:
		log.Errorf("Kubernetes Error: %v", kerr.Error())
		err = kerr
	}

	return err
}

func getPodObject(paramsStr string, params PiquantParams, dockerImage string, jobid, namespace string, creator pixlUser.UserInfo, length int) *apiv1.Pod {
	sec := apiv1.LocalObjectReference{Name: "api-auth"}
	parts := strings.Split(params.PMCListName, ".")
	return &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobid + "-" + parts[0],
			Namespace: namespace,
			Labels: map[string]string{
				"app":          parts[0],
				"owner":        creator.UserID,
				"jobid":        jobid,
				"numberofpods": strconv.Itoa(length),
			},
		},
		Spec: apiv1.PodSpec{
			ImagePullSecrets: []apiv1.LocalObjectReference{sec},
			RestartPolicy:    apiv1.RestartPolicyNever,

			Containers: []apiv1.Container{
				{
					Name:            parts[0],
					Image:           dockerImage,
					ImagePullPolicy: apiv1.PullAlways,
					Resources: apiv1.ResourceRequirements{
						Requests: apiv1.ResourceList{
							"cpu": resource.MustParse("3500m"),
						},
					},

					Env: []apiv1.EnvVar{
						{Name: "QUANT_PARAMS", Value: paramsStr},
						{Name: "AWS_ACCESS_KEY_ID", Value: os.Getenv("AWS_ACCESS_KEY_ID")},
						{Name: "AWS_SECRET_ACCESS_KEY", Value: os.Getenv("AWS_SECRET_ACCESS_KEY")},
						{Name: "AWS_DEFAULT_REGION", Value: os.Getenv("AWS_DEFAULT_REGION")},
						{Name: "PYTHONUNBUFFERED", Value: "TRUE"},
					},
				},
			},
		},
	}
}

func runQuantJob(wg *sync.WaitGroup, params PiquantParams, jobid string, namespace string, dockerImage string, creator pixlUser.UserInfo, count int, log logger.ILogger) {
	defer wg.Done()

	// Make a JSON string out of params so it can be passed in
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		log.Errorf("Failed to serialise JSON params for node: %v", params.PMCListName)
		return
	}
	paramsStr := string(paramsJSON)

	//log.Debugf("getPodObject for: %v. namespace: %v, count: %v", params.PMCListName, namespace, count)
	pod := getPodObject(paramsStr, params, dockerImage, jobid, namespace, creator, count)

	co := metav1.CreateOptions{}
	pod, err = kubeHelper.Clientset.CoreV1().Pods(pod.Namespace).Create(context.TODO(), pod, co)
	if err != nil {
		log.Errorf("Pod create failed for: %v. namespace: %v, count: %v", params.PMCListName, namespace, count)
		fatalErrors <- err
		return
	}

	// Create Deployment
	log.Debugf("Creating pod for %v in namespace %v...", params.PMCListName, namespace)

	// Now wait for it to finish
	startUnix := time.Now().Unix()
	maxEndUnix := startUnix + config.KubernetesMaxTimeoutSec

	lastPhase := ""

	for currUnix := time.Now().Unix(); currUnix < maxEndUnix; currUnix = time.Now().Unix() {
		// Check kubernetes pod status
		pod, _ := kubeHelper.Clientset.CoreV1().Pods(pod.Namespace).Get(context.TODO(), pod.Name, metav1.GetOptions{})

		// TODO: is this needed, now that we log?
		//fmt.Println(pod.Status.Phase)
		//log.Infof("%v phase: %v, pod name: %v, namespace: %v", params.PMCListName, pod.Status.Phase, pod.Name, pod.Namespace)

		phase := ""
		phase = string(pod.Status.Phase)
		if lastPhase != phase {
			log.Debugf("%v phase: %v, pod name: %v, namespace: %v", params.PMCListName, pod.Status.Phase, pod.Name, pod.Namespace)
			lastPhase = phase
		}

		if pod.Status.Phase != apiv1.PodRunning && pod.Status.Phase != apiv1.PodPending {
			log.Debugf("Deleting pod: %v from namespace: %v", pod.Name, pod.Namespace)

			deletePolicy := metav1.DeletePropagationForeground
			do := &metav1.DeleteOptions{
				PropagationPolicy: &deletePolicy,
			}
			err := kubeHelper.Clientset.CoreV1().Pods(pod.Namespace).Delete(context.TODO(), pod.Name, *do)
			if err != nil {
				log.Errorf("Failed to remove pod: %v, namespace: %v\n", pod.Name, pod.Namespace)
			}
			break
		}

		time.Sleep(5 * time.Second)
	}
}
