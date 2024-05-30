package rbac

import (
	"testing"

	"github.com/syndicut/timonify/pkg/metadata"

	"github.com/stretchr/testify/assert"
	"github.com/syndicut/timonify/internal"
)

const clusterRoleYaml = `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  creationTimestamp: null
  name: my-operator-manager-role
aggregationRule:
  clusterRoleSelectors:
  - matchExpressions:
    - key: my.operator.dev/release
      operator: Exists
rules:
- apiGroups:
  - ""
  resources:
  - pods
  verbs:
  - get
  - list`

func Test_clusterRole_Process(t *testing.T) {
	var testInstance role

	t.Run("processed", func(t *testing.T) {
		obj := internal.GenerateObj(clusterRoleYaml)
		processed, _, err := testInstance.Process(&metadata.Service{}, obj)
		assert.NoError(t, err)
		assert.Equal(t, true, processed)
	})
	t.Run("skipped", func(t *testing.T) {
		obj := internal.TestNs
		processed, _, err := testInstance.Process(&metadata.Service{}, obj)
		assert.NoError(t, err)
		assert.Equal(t, false, processed)
	})
}
