package simple_k8s

import (
	"errors"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	Functional "github.com/geniussportsgroup/FunctionalLib"
	List "github.com/geniussportsgroup/Slist"
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

	cmpStr := func(i1, i2 interface{}) bool {
		return i1.(string) < i2.(string)
	}

	ret := Set.NewTreap(cmpStr)
	foundClues := Set.NewTreap(cmpStr)

	for _, item := range list.Items {
		for _, clue := range clues {
			if strings.Contains(item.ObjectMeta.Name, clue.(string)) {
				ret.Insert(item.ObjectMeta.Name)
				foundClues.Insert(clue)
				break
			}
		}
	}

	// check that all the clues were found in the deployment names
	for _, clue := range clues {
		if foundClues.Search(clue) == nil {
			return nil, errors.New(fmt.Sprintf("Deployment name containing clue %s was not found", clue))
		}
	}

	return ret, nil
}

// Return a list of pair <clue, deployName> containing all the found namespaces whose name contains any given clue as substring.
func ReadDeploymentNames(kubectl *kubernetes.Clientset, kubeNamespace, labelSelector string,
	clues *List.Slist) (*List.Slist, error) {

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

	cmpStr := func(i1, i2 interface{}) bool {
		return i1.(string) < i2.(string)
	}

	ret := List.New()
	foundClues := Set.NewTreap(cmpStr)

	for _, item := range list.Items {
		for it := List.NewIterator(clues); it.HasCurr(); it.Next() {
			clue := it.GetCurr().(string)
			if strings.Contains(item.ObjectMeta.Name, clue) {
				ret.Append(Functional.Pair{Item1: clue, Item2: item.ObjectMeta.Name})
				foundClues.Insert(clue)
				break
			}
		}
	}

	// check that all the clues were found in the deployment names
	for it := List.NewIterator(clues); it.HasCurr(); it.Next() {
		clue := it.GetCurr().(string)
		if foundClues.Search(clue) == nil {
			return nil, errors.New(fmt.Sprintf("Deployment name containing clue %s was not found", clue))
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

// goroutine for handling SIGTERM
func TerminationHandler(timeout time.Duration) {

	log.Printf("Setting termination handler for process pid = %d", os.Getpid())
	signChannel := make(chan os.Signal, 1)

	signal.Ignore(syscall.SIGINT, syscall.SIGQUIT, syscall.SIGCONT)
	signal.Notify(signChannel, syscall.SIGTERM)

	sig := <-signChannel

	log.Printf("Termination received (%s)", sig.String())

	time.Sleep(timeout)

	os.Exit(0)
}
