/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"bytes"
	"context"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strconv"

	"github.com/go-logr/logr"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	jobsv1alpha1 "gihub.com/amirhmi/kubebuilder/api/v1alpha1"
)

// PeriodicCurlReconciler reconciles a PeriodicCurl object
type PeriodicCurlReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=jobs.amir.job,resources=periodiccurls,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=jobs.amir.job,resources=periodiccurls/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods/status,verbs=get

func createPodFromSpec(pcSpec jobsv1alpha1.PeriodicCurlSpec, namespace string, sleepingAmount string) core.Pod {
	initSleepCommand := "sleep " + sleepingAmount + ";"
	curlCommand := initSleepCommand + "curl -o /dev/null -s -w '%{time_total}\n' " + pcSpec.Url
	commands := []string{"/bin/bash", "-c", curlCommand}

	newPod := core.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-pod",
			Namespace: namespace,
		},
		Spec: core.PodSpec{
			RestartPolicy: "Never",
			Containers: []core.Container{
				{
					Name:    "url-caller",
					Image:   "quay.io/amirhmi/ubuntu-curl",
					Command: commands,
				},
			},
		},
	}

	return newPod
}

func getPodLogs(pod *core.Pod) string {
	podLogOpts := core.PodLogOptions{}
	// use the current context in kubeconfig
	const KubeConfigPath = "/home/amirhmi/.kube/config"
	config, err := clientcmd.BuildConfigFromFlags("", KubeConfigPath)
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "error in getting access to K8S"
	}
	req := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &podLogOpts)
	podLogs, err := req.Stream()
	if err != nil {
		return "error in opening stream"
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "error in copy information from podLogs to buf"
	}
	str := buf.String()

	return str
}

func getChildPod(r *PeriodicCurlReconciler, ctx context.Context, namespace string) (*core.Pod, error) {
	var childPods core.PodList
	if err := r.List(ctx, &childPods, client.InNamespace(namespace)); err != nil {
		return &core.Pod{}, err
	}
	if len(childPods.Items) > 0 {
		return &childPods.Items[0], nil
	} else {
		return nil, nil
	}
}

func isContainerOfPodSucceeded(pod *core.Pod) bool {
	const completedContainerMessage = "Completed"
	for _, c := range pod.Status.ContainerStatuses {
		if c.State.Terminated != nil && c.State.Terminated.Reason == completedContainerMessage {
			return true
		}
	}
	return false
}

func (r *PeriodicCurlReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("periodiccurl", req.NamespacedName)

	var periodicCurl = &jobsv1alpha1.PeriodicCurl{}
	if err := r.Get(ctx, req.NamespacedName, periodicCurl); err != nil {
		log.Error(err, "unable to fetch PeriodicCurl")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	childPod, err := getChildPod(r, ctx, req.Namespace)
	if err != nil {
		log.Error(err, "unable to get child pod")
		return ctrl.Result{}, err
	}
	if childPod != nil {
		if isContainerOfPodSucceeded(childPod) {
			curlResponseTimeLog := getPodLogs(childPod)
			curlResponseTime, err := strconv.ParseFloat(curlResponseTimeLog[:len(curlResponseTimeLog)-1], 64)
			if err == nil {
				nextCurlTime := curlResponseTime * float64(periodicCurl.Spec.Coefficient)
				periodicCurl.Status.NextCurlTime = strconv.FormatFloat(nextCurlTime, 'E', -1, 64)
				r.Status().Update(ctx, periodicCurl)
				log.Info("curl response time: " + curlResponseTimeLog)
				r.Delete(ctx, childPod)
			}
		}
	} else {
		log.Info("next curl time status: " + periodicCurl.Status.NextCurlTime)
		newPod := createPodFromSpec(periodicCurl.Spec, periodicCurl.Namespace, periodicCurl.Status.NextCurlTime)
		if err := controllerutil.SetControllerReference(periodicCurl, &newPod, r.Scheme); err != nil {
			log.Error(err, "unable to set reference for the new pod")
			return ctrl.Result{}, err
		}
		if err := r.Create(ctx, &newPod); err != nil {
			log.Error(err, "unable to create new pod")
			return ctrl.Result{}, err
		}
		log.Info("new pod created")
	}

	return ctrl.Result{}, nil
}

func (r *PeriodicCurlReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&jobsv1alpha1.PeriodicCurl{}).
		Owns(&core.Pod{}).
		Complete(r)
}
