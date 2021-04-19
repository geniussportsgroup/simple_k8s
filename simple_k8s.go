package simple_k8s

import (
	"errors"
	"fmt"
	"github.com/geniussportsgroup/FunctionalLib"
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

	List "github.com/geniussportsgroup/Slist"
	Set "github.com/geniussportsgroup/treaps"
)

const HealthyFileName = "/tmp/healthy"

func CreateHealthyFile(name string) {
	os.OpenFile(name, os.O_RDONLY|os.O_CREATE, 0666)
}

func RemoveHealthyFile(name string) {
	_ = os.Remove(name)
}

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

// FindDeploymentNames Return a set containing all the found namespaces whose name contains any given clue as substring.
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

// ReadDeploymentNames Return a list of pair <clue, deployName> containing all the found namespaces whose name contains any given clue as substring.
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

	cluesToDeployName := make(map[string]string)

	for _, item := range list.Items {
		for it := List.NewIterator(clues); it.HasCurr(); it.Next() {
			clue := it.GetCurr().(string)
			if strings.Contains(item.ObjectMeta.Name, clue) {
				cluesToDeployName[clue] = item.ObjectMeta.Name
				break
			}
		}
	}

	// check that all the clues were found in the deployment names
	for it := List.NewIterator(clues); it.HasCurr(); it.Next() {
		clue := it.GetCurr().(string)
		if _, found := cluesToDeployName[clue]; !found {
			return nil, errors.New(fmt.Sprintf("Deployment name containing clue %s was not found", clue))
		}
	}

	return FunctionalLib.Map(clues, func(i interface{}) interface{} {
		clue := i.(string)
		return FunctionalLib.Pair{
			Item1: clue,
			Item2: cluesToDeployName[clue],
		}
	}), nil
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

// TerminationHandler goroutine for handling SIGTERM
func TerminationHandler(timeout time.Duration) {

	TerminationHandlerCont(timeout, nil)
}

// TerminationHandlerCont set a termination handler. When SIGTERM is received, waits for timeout. Next
// the continuation function is called with the parameters pars... (it is up programmer to perform the casting).
// continuation is a point for giving to the programmer some additional actions that they could require
// before to terminate the process
func TerminationHandlerCont(timeout time.Duration, continuation func(pars ...interface{}), pars ...interface{}) {

	log.Printf("Setting termination handler for process pid = %d", os.Getpid())
	signChannel := make(chan os.Signal, 1)

	signal.Ignore(syscall.SIGINT, syscall.SIGQUIT, syscall.SIGCONT)
	signal.Notify(signChannel, syscall.SIGTERM)

	sig := <-signChannel

	log.Printf("Termination received (%s)", sig.String())

	if continuation != nil {
		continuation(pars...)
	}

	log.Printf("Waiting for %s seconds before to exit", timeout/time.Second)
	time.Sleep(timeout)
	log.Print("Exiting")
	os.Exit(0)
}

// SetTerminationHandler Wrapper for setting the goroutine prepared for handling the SIGTERM
func SetTerminationHandler(TerminationTimeout time.Duration) {
	go TerminationHandler(TerminationTimeout)
}

// SetTerminationHandlerWithContinuation Wrapper for setting the goroutine prepared for handling the SIGTERM
func SetTerminationHandlerWithContinuation(TerminationTimeout time.Duration,
	continuation func(pars ...interface{}), pars ...interface{}) {
	go TerminationHandlerCont(TerminationTimeout, continuation, pars...)
}
