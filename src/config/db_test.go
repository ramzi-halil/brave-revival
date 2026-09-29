package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"example.com/brave-revival/src/config"
	"example.com/brave-revival/src/proto/pmaster"
	"github.com/stretchr/testify/require"
)

const sampleResources = `id,ios,android,ios_size,android_size
1001004,71901152b0d58ee3e2b4a061bf602598,7eb845a36ca56abec480c4cad21399c7,123,456
380026268,930f80a03ebfaac91f620e329d862fba,6fa628316f11fa42199863fdd1161d80,789,1011
98020335,e5ba00936d24de444f9f16c7cbc43615,e5ba00936d24de444f9f16c7cbc43615,1213,1213
`

func TestLoadResources(t *testing.T) {
	dbDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dbDir, "resources.csv"), []byte(sampleResources), 0o600))
	actual, err := config.LoadResources(dbDir)

	require.NoError(t, err)
	require.Equal(t, config.Resources{
		"ios": {
			Resource: map[uint32]*pmaster.ResourceInfo{
				1001004:   &pmaster.ResourceInfo{Hash: "71901152b0d58ee3e2b4a061bf602598", Size: 123},
				380026268: &pmaster.ResourceInfo{Hash: "930f80a03ebfaac91f620e329d862fba", Size: 789},
				98020335:  &pmaster.ResourceInfo{Hash: "e5ba00936d24de444f9f16c7cbc43615", Size: 1213},
			},
		},
		"android": &pmaster.Resources{
			Resource: map[uint32]*pmaster.ResourceInfo{
				1001004:   &pmaster.ResourceInfo{Hash: "7eb845a36ca56abec480c4cad21399c7", Size: 456},
				380026268: &pmaster.ResourceInfo{Hash: "6fa628316f11fa42199863fdd1161d80", Size: 1011},
				98020335:  &pmaster.ResourceInfo{Hash: "e5ba00936d24de444f9f16c7cbc43615", Size: 1213},
			},
		},
	}, actual)
}
