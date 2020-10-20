package simple_k8s

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"strings"

	Set "github.com/geniussportsgroup/treaps"
)

func NewKubernetesClient(pathToConf string) (*kubernetes.Clientset, error) {
	var config *rest.Config
	var err error
	if pathToConf == "" {
		config, err = rest.InClusterConfig()
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", pathToConf)
	}
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

// Return a set containing all the found namespaces whose name contains any given clue as substring.
func FindDeploymentNames(kubectl *kubernetes.Clientset, kubeNamespace, labelSelector string,
	clues ...interface{}) (*Set.Treap, error) {

	list, err := kubectl.AppsV1().Deployments(kubeNamespace).List(metav1.ListOptions{
		TypeMeta:            metav1.TypeMeta{},
		LabelSelector:       labelSelector,
		FieldSelector:       "",
		Watch:               false,
		AllowWatchBookmarks: false,
		ResourceVersion:     "",
		TimeoutSeconds:      nil,
		Limit:               0,
		Continue:            "",
	})
	if err != nil {
		return nil, err
	}

	ret := Set.NewTreap(func(i1, i2 interface{}) bool {
		return i1.(string) < i2.(string)
	})

	for _, item := range list.Items {
		for _, clue := range clues {
			if strings.Contains(item.ObjectMeta.Name, clue.(string)) {
				ret.Insert(item.ObjectMeta.Name)
				break
			}
		}
	}

	return ret, nil
}

func GetNumberOfPods(kubectl *kubernetes.Clientset, kubeNamespace string,
	deploymentName string) (n int32, err error) {

	result, err := kubectl.AppsV1().Deployments(kubeNamespace).
		GetScale(deploymentName, metav1.GetOptions{})
	if err != nil {
		return
	}

	n = result.Spec.Replicas

	return
}

func SetNumberOfPods(numPods int32, currentNumOfPods *int32,
	kubectl *kubernetes.Clientset, kubeNamespace string, deploymentName string) (bool, error) {

	// Consult the current number of pods under the deployment
	scale, err := kubectl.AppsV1().Deployments(kubeNamespace).
		GetScale(deploymentName, metav1.GetOptions{})
	if err != nil {
		return false, err
	}

	*currentNumOfPods = scale.Spec.Replicas
	if *currentNumOfPods == numPods {
		return false, nil
	}

	// Set new scale
	scale.Spec.Replicas = numPods
	scale, err = kubectl.AppsV1().
		Deployments(kubeNamespace).
		UpdateScale(deploymentName, scale)
	if err != nil {
		return false, err
	}

	*currentNumOfPods = scale.Spec.Replicas

	return true, nil
}
