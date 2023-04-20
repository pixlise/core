package kubernetes

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"os"
	"reflect"
	"sync"
	"time"

	"github.com/pixlise/core/v3/api/config"
	"github.com/pixlise/core/v3/core/logger"
	"github.com/pixlise/core/v3/core/pixlUser"
	"github.com/pixlise/core/v3/core/utils"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var fatalErrors chan error
var wgPod chan string

type KubeHelper struct {
	Clientset  *kubernetes.Clientset
	Kubeconfig string
	Log        logger.ILogger
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func (k *KubeHelper) Bootstrap(location string, apiLog logger.ILogger) {
	// We'll need the logger later...
	k.Log = apiLog
	var err error

	// Don't run multiple times (?)
	if k.Clientset != nil && !reflect.ValueOf(k.Clientset.CoreV1()).IsNil() {
		k.Log.Infof("KubeHelper Bootstrap not run...")
		return
	}

	// Decide if internal or external kubernetes
	var conf *rest.Config

	if location == "external" {
		k.Log.Debugf("Bootstrapping kubernetes as external")

		// use the current context in kubeconfig
		conf, err = clientcmd.BuildConfigFromFlags("", k.Kubeconfig)
		if err != nil {
			k.Log.Errorf("Kubernetes BuildConfigFromFlags error: %v", err.Error())
		}
	} else {
		k.Log.Debugf("Bootstrapping kubernetes as internal")

		conf, err = rest.InClusterConfig()
		if err != nil {
			k.Log.Errorf("Kubernetes InClusterConfig failed: %v", err.Error())
		}
	}

	clientset := &kubernetes.Clientset{}
	clientset, err = kubernetes.NewForConfig(conf)
	if err != nil {
		k.Log.Errorf("Kubernetes NewForConfig failed: %v", err.Error())
	}
	/* Took this out because it was erroring due to permissions - we should probably be querying just for this namespace.
	Since it didn't do anything with the number queried, we don't need it...

	pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		k.Log.Errorf("Failed to list pods: %v", err.Error())
	}
	if pods == nil {
		k.Log.Errorf("Pod list returned is nil")
	} else {
		k.Log.Infof("There are %d pods in the cluster", len(pods.Items))
	}
	*/
	k.Clientset = clientset
}

func (k *KubeHelper) RunPod(
	cmd []string,
	args []string,
	env map[string]string,
	volumes []apiv1.Volume,
	volumeMounts []apiv1.VolumeMount,
	dockerImage string,
	namespace string,
	podnameprefix string,
	labels map[string]string,
	creator pixlUser.UserInfo,
	background bool) (string, error) {

	var err error
	var wg sync.WaitGroup
	fatalErrors = make(chan error)
	wgDone := make(chan bool)
	wgPod = make(chan string, 2)
	wg.Add(1)
	go k.launchPod(&wg, cmd, args, env, volumes, volumeMounts, dockerImage, namespace, podnameprefix, labels, creator, background)
	// Wait for all piquant instances to finish
	//wg.Wait()
	go func() {
		wg.Wait()
		close(wgDone)
	}()

	k.Log.Infof("Waiting for pod...")

	podname := ""
	select {
	case <-wgDone:
		k.Log.Infof("Kubernetes pods reported complete")
		msg := <-wgPod
		podname = fmt.Sprintf("%v", msg)
		break
	case kerr := <-fatalErrors:
		k.Log.Errorf("Kubernetes Error: %v", kerr.Error())
		err = kerr
	}

	return podname, err
}

func (k *KubeHelper) launchPod(wg *sync.WaitGroup, cmd []string, args []string, env map[string]string, volumes []apiv1.Volume, volumeMounts []apiv1.VolumeMount, dockerImage string, namespace string, podnameprefix string, labels map[string]string, creator pixlUser.UserInfo, background bool) {
	defer wg.Done()

	podobj := k.getPodObject(cmd, args, env, volumes, volumeMounts, dockerImage, namespace, podnameprefix, labels, creator)

	co := metav1.CreateOptions{}
	pod, err := k.Clientset.CoreV1().Pods(podobj.Namespace).Create(context.TODO(), podobj, co)
	if err != nil {
		k.Log.Errorf("Pod create failed for: %v. namespace: %v.", podnameprefix, namespace)
		fatalErrors <- err
	}

	// Create Deployment
	k.Log.Debugf("Creating pod for %v in namespace %v...", pod.Name, namespace)

	// Now wait for it to finish
	startUnix := time.Now().Unix()
	maxEndUnix := startUnix + config.KubernetesMaxTimeoutSec

	lastPhase := ""

	for currUnix := time.Now().Unix(); currUnix < maxEndUnix; currUnix = time.Now().Unix() {
		// Check kubernetes pod status
		pod, _ := k.Clientset.CoreV1().Pods(pod.Namespace).Get(context.TODO(), pod.Name, metav1.GetOptions{})

		k.Log.Infof("%v phase: %v, namespace: %v", pod.Name, pod.Status.Phase, pod.Namespace)

		phase := string(pod.Status.Phase)
		if lastPhase != phase {
			k.Log.Debugf("%v phase: %v, namespace: %v", pod.Name, pod.Status.Phase, pod.Namespace)
			lastPhase = phase
		}

		if (!background && pod.Status.Phase != apiv1.PodRunning && pod.Status.Phase != apiv1.PodPending) ||
			(background && pod.Status.Phase != apiv1.PodPending) {
			k.Log.Debugf("Deleting pod: %v from namespace: %v", pod.Name, pod.Namespace)
			if pod.Status.Phase == apiv1.PodFailed {
				logs := k.getPodLogs(*pod)
				k.Log.Errorf("Pod Logs: %v", logs)
				fatalErrors <- fmt.Errorf("Pod %v, failed to complete", pod.Name)
			}
			if pod.Status.Phase != apiv1.PodRunning {
				err = k.DeletePod(pod.Namespace, pod.Name)
				if err != nil {
					k.Log.Errorf("Failed to remove pod: %v, namespace: %v\n", pod.Name, pod.Namespace)
					fatalErrors <- fmt.Errorf("Failed to remove pod: %v, namespace: %v\n", pod.Name, pod.Namespace)
				}
			}
			wgPod <- pod.Name
			break
		}
		time.Sleep(5 * time.Second)
	}
}

func (k *KubeHelper) getPodLogs(pod apiv1.Pod) string {
	podLogOpts := apiv1.PodLogOptions{}
	req := k.Clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &podLogOpts)
	podLogs, err := req.Stream(context.TODO())
	if err != nil {
		return "error in opening pod log stream"
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "error copying log from pod"
	}

	return buf.String()
}

func (k *KubeHelper) DeletePod(namespace string, name string) error {
	deletePolicy := metav1.DeletePropagationForeground
	do := &metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}
	return k.Clientset.CoreV1().Pods(namespace).Delete(context.TODO(), name, *do)
}

func (k *KubeHelper) getPodObject(cmd []string, args []string, env map[string]string, volumes []apiv1.Volume, volumeMounts []apiv1.VolumeMount, dockerImage string, namespace string, podnameprefix string, labels map[string]string, creator pixlUser.UserInfo) *apiv1.Pod {
	sec := apiv1.LocalObjectReference{Name: "api-auth"}
	rand.Seed(time.Now().UnixNano())
	podname := podnameprefix + utils.RandStringBytesMaskImpr(16)
	labels["owner"] = creator.UserID
	optional := true
	envvar := []apiv1.EnvVar{
		{Name: "PYTHONUNBUFFERED", Value: "TRUE"},
		{ // TODO: Remove this or change how it is injected once we pick the ocs-poster work back up
			Name: "credss_password",
			ValueFrom: &apiv1.EnvVarSource{
				SecretKeyRef: &apiv1.SecretKeySelector{
					LocalObjectReference: apiv1.LocalObjectReference{
						Name: "m20-sstage-secret",
					},
					Key:      "password",
					Optional: &optional,
				},
			},
		},
	}
	for key, element := range env {
		envvar = append(envvar, apiv1.EnvVar{Name: key, Value: element})
	}
	return &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podname,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: apiv1.PodSpec{
			ImagePullSecrets: []apiv1.LocalObjectReference{sec},
			RestartPolicy:    apiv1.RestartPolicyNever,
			Volumes:          volumes,
			Containers: []apiv1.Container{
				{
					Name:            podname,
					Image:           dockerImage,
					ImagePullPolicy: apiv1.PullAlways,
					Resources: apiv1.ResourceRequirements{
						Requests: apiv1.ResourceList{
							"cpu": resource.MustParse("3500m"),
						},
					},
					Command:      cmd,
					Args:         args,
					VolumeMounts: volumeMounts,
					Env:          envvar,
				},
			},
		},
	}
}
