package processor

import (
	"github.com/syndicut/timonify/pkg/config"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/syndicut/timonify/internal"
	"github.com/syndicut/timonify/pkg/metadata"
)

func TestProcessObjMeta(t *testing.T) {
	testMeta := metadata.New(config.Config{ModuleName: "module-name"})
	testMeta.Load(internal.TestNs)
	res, err := ProcessObjMeta(testMeta, internal.TestNs)
	assert.NoError(t, err)
	assert.Contains(t, res, "module-name.labels")
	assert.Contains(t, res, "module-name.fullname")
}
