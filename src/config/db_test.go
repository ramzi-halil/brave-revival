package config_test

import (
	"strings"
	"testing"

	"example.com/brave-revival/src/config"
	"example.com/brave-revival/src/proto/pmaster"
	"github.com/stretchr/testify/require"
)

const sampleResources = `id,ios,android
1001004,71901152b0d58ee3e2b4a061bf602598,7eb845a36ca56abec480c4cad21399c7
380026268,930f80a03ebfaac91f620e329d862fba,6fa628316f11fa42199863fdd1161d80
98020335,e5ba00936d24de444f9f16c7cbc43615,e5ba00936d24de444f9f16c7cbc43615
`

func TestLoadResources(t *testing.T) {
	f := strings.NewReader(sampleResources)
	actual, err := config.LoadResources(f)

	require.NoError(t, err)
	require.Equal(t, config.Resources{
		"ios": {
			Resource: map[uint32]*pmaster.ResourceInfo{
				1001004:   &pmaster.ResourceInfo{Hash: "71901152b0d58ee3e2b4a061bf602598"},
				380026268: &pmaster.ResourceInfo{Hash: "930f80a03ebfaac91f620e329d862fba"},
				98020335:  &pmaster.ResourceInfo{Hash: "e5ba00936d24de444f9f16c7cbc43615"},
			},
		},
		"android": &pmaster.Resources{
			Resource: map[uint32]*pmaster.ResourceInfo{
				1001004:   &pmaster.ResourceInfo{Hash: "7eb845a36ca56abec480c4cad21399c7"},
				380026268: &pmaster.ResourceInfo{Hash: "6fa628316f11fa42199863fdd1161d80"},
				98020335:  &pmaster.ResourceInfo{Hash: "e5ba00936d24de444f9f16c7cbc43615"},
			},
		},
	}, actual)
}
