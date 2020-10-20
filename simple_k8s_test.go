package simple_k8s

import (
	"fmt"
	Set "github.com/geniussportsgroup/treaps"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

const TestPathToKubeConfig = "../../.kube/config"

func TestNewKubernetesClient(t *testing.T) {
	kubectl, err := NewKubernetesClient(TestPathToKubeConfig)
	assert.Nil(t, err)
	assert.NotNil(t, kubectl)
}

func Test_findDeploymentNames(t *testing.T) {
	kubectl, err := NewKubernetesClient(TestPathToKubeConfig)
	assert.Nil(t, err)
	assert.NotNil(t, kubectl)

	l, err := FindDeploymentNames(kubectl, "basketball-nba-uat", "versionsvc=2-2-3",
		"score-diff", "score-total", "home-away", "race", "team-score-last",
		"freethrow", "ponderate")

	for it := Set.NewIterator(l); it.HasCurr(); it.Next() {
		fmt.Println(it.GetCurr().(string))
		assert.NotNil(t, l.Search(it.GetCurr()))
	}
}

func TestGetNumberOfPods(t *testing.T) {

	kubectl, err := NewKubernetesClient(TestPathToKubeConfig)
	assert.Nil(t, err)
	assert.NotNil(t, kubectl)

	l, err := FindDeploymentNames(kubectl, "basketball-nba-uat", "versionsvc=2-2-3",
		"score-diff", "score-total", "home-away", "race", "team-score-last",
		"freethrow", "ponderate")

	for it := Set.NewIterator(l); it.HasCurr(); it.Next() {
		deployName := it.GetCurr().(string)
		numPods, err := GetNumberOfPods(kubectl, "basketball-nba-uat", deployName)
		assert.Nil(t, err)
		fmt.Println(deployName, " =", numPods)
	}
}

func TestSetNumberOfPods(t *testing.T) {

	kubectl, err := NewKubernetesClient(TestPathToKubeConfig)
	assert.Nil(t, err)
	assert.NotNil(t, kubectl)

	fmt.Println("Reading deployment names")
	l, err := FindDeploymentNames(kubectl, "basketball-nba-uat", "versionsvc=2-2-3",
		"score-diff", "score-total", "home-away", "race", "team-score-last",
		"freethrow", "ponderate")

	fmt.Println("These are the deploys")
	for it := Set.NewIterator(l); it.HasCurr(); it.Next() {
		fmt.Println("    ", it.GetCurr())
	}

	fmt.Println("Reading number of pods per deploy")
	for it := Set.NewIterator(l); it.HasCurr(); it.Next() {
		deployName := it.GetCurr().(string)
		numPods, err := GetNumberOfPods(kubectl, "basketball-nba-uat", deployName)
		ok, err := SetNumberOfPods(numPods+1, &numPods, kubectl, "basketball-nba-uat", deployName)
		assert.True(t, ok)
		assert.Nil(t, err)
		fmt.Println(deployName, " =", numPods)
	}

	fmt.Println("Scaling every deploy in one pod")
	for it := Set.NewIterator(l); it.HasCurr(); it.Next() {
		deployName := it.GetCurr().(string)
		numPods, err := GetNumberOfPods(kubectl, "basketball-nba-uat", deployName)
		assert.Nil(t, err)
		fmt.Println(deployName, " =", numPods)
	}

	fmt.Println("Waiting 30 second before to scale down")
	time.Sleep(30 * time.Second)
	fmt.Println("Scaling down")

	for it := Set.NewIterator(l); it.HasCurr(); it.Next() {
		deployName := it.GetCurr().(string)
		numPods, err := GetNumberOfPods(kubectl, "basketball-nba-uat", deployName)
		ok, err := SetNumberOfPods(numPods-1, &numPods, kubectl, "basketball-nba-uat", deployName)
		assert.True(t, ok)
		assert.Nil(t, err)
		fmt.Println(deployName, " =", numPods)
	}
}
