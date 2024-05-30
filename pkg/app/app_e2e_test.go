package app

import (
	"bufio"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/syndicut/timonify/pkg/config"
	"helm.sh/helm/v3/pkg/action"
)

const (
	operatorChartName = "test-operator"
	appChartName      = "test-app"
)

func TestOperator(t *testing.T) {
	file, err := os.Open("../../test_data/k8s-operator-kustomize.output")
	assert.NoError(t, err)

	err = os.RemoveAll(operatorChartName)
	objects := bufio.NewReader(file)
	err = Start(objects, config.Config{ChartName: operatorChartName})
	assert.NoError(t, err)

	//t.Cleanup(func() {
	//	err = os.RemoveAll(operatorChartName)
	//	assert.NoError(t, err)
	//})

	timoniLint := exec.Command("timoni", "--namespace", "test-ns", "mod", "lint", operatorChartName)
	err = timoniLint.Run()
	assert.NoError(t, err)
}

func TestApp(t *testing.T) {
	file, err := os.Open("../../test_data/sample-app.yaml")
	assert.NoError(t, err)

	objects := bufio.NewReader(file)
	err = Start(objects, config.Config{ChartName: appChartName})
	assert.NoError(t, err)

	t.Cleanup(func() {
		err = os.RemoveAll(appChartName)
		assert.NoError(t, err)
	})

	helmLint := action.NewLint()
	helmLint.Strict = true
	helmLint.Namespace = "test-ns"
	result := helmLint.Run([]string{appChartName}, nil)
	for _, err = range result.Errors {
		assert.NoError(t, err)
	}
}
