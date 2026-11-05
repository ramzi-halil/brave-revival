package config_test

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"example.com/brave-revival/src/config"
	"example.com/brave-revival/src/proto/pmaster"
	"github.com/stretchr/testify/require"
	gproto "google.golang.org/protobuf/proto"
)

const sampleVersion = "platform,application,resource,master\nandroid,1.44.274,6749,12809\n"

func TestLoadMasterCache(t *testing.T) {
	for _, state := range []string{"missing", "invalid gzip", "invalid protobuf", "truncated gzip", "stale", "unwritable"} {
		t.Run(state, func(t *testing.T) {
			dbDir := t.TempDir()
			fields := (&pmaster.All{}).ProtoReflect().Descriptor().Fields()
			for i := range fields.Len() {
				field := fields.Get(i)
				columns := field.Message().Fields()
				header := make([]string, columns.Len())
				for j := range columns.Len() {
					header[j] = string(columns.Get(j).Name())
				}
				require.NoError(t, os.WriteFile(filepath.Join(dbDir, string(field.Name())+".csv"), []byte(strings.Join(header, ",")+"\n"), 0o600))
			}
			require.NoError(t, os.WriteFile(filepath.Join(dbDir, "version.csv"), []byte(sampleVersion), 0o600))
			cachePath := filepath.Join(dbDir, "__all.pb.gz")
			switch state {
			case "invalid gzip":
				require.NoError(t, os.WriteFile(cachePath, []byte("invalid"), 0o600))
			case "invalid protobuf":
				writeMasterTestCache(t, cachePath, []byte{0xff})
			case "truncated gzip", "stale":
				data, err := gproto.Marshal(&pmaster.All{Version: []*pmaster.Version{{Platform: "android", Master: 1}}})
				require.NoError(t, err)
				writeMasterTestCache(t, cachePath, data)
				if state == "truncated gzip" {
					info, err := os.Stat(cachePath)
					require.NoError(t, err)
					require.NoError(t, os.Truncate(cachePath, info.Size()-8))
				}
			case "unwritable":
				require.NoError(t, os.Mkdir(cachePath, 0o700))
			}

			all, err := config.LoadMaster(dbDir)
			require.NoError(t, err)
			require.Len(t, all.Version, 1)
			require.Equal(t, uint32(12809), all.Version[0].Master)
			if state == "unwritable" {
				return
			}

			f, err := os.Open(cachePath)
			require.NoError(t, err)
			defer f.Close()
			reader, err := gzip.NewReader(f)
			require.NoError(t, err)
			defer reader.Close()
			data, err := io.ReadAll(reader)
			require.NoError(t, err)
			cached := &pmaster.All{}
			require.NoError(t, gproto.Unmarshal(data, cached))
			require.True(t, gproto.Equal(all, cached))

			require.NoError(t, os.Remove(filepath.Join(dbDir, "achievement.csv")))
			loaded, err := config.LoadMaster(dbDir)
			require.NoError(t, err)
			require.True(t, gproto.Equal(all, loaded))

			require.NoError(t, os.WriteFile(filepath.Join(dbDir, "version.csv"), []byte("invalid\n"), 0o600))
			_, err = config.LoadMaster(dbDir)
			require.Error(t, err)
		})
	}
}

func writeMasterTestCache(t *testing.T, path string, data []byte) {
	t.Helper()
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	_, err := writer.Write(data)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	require.NoError(t, os.WriteFile(path, compressed.Bytes(), 0o600))
}

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
