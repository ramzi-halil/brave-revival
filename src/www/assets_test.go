package www

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenResourceByHash(t *testing.T) {
	h := &Handler{}
	dirs := []string{t.TempDir(), t.TempDir()}
	for _, dir := range dirs {
		root, err := os.OpenRoot(dir)
		require.NoError(t, err)
		t.Cleanup(func() { root.Close() })
		h.assets = append(h.assets, root)
	}

	const hash = "abcdef"
	firstPath := filepath.Join(dirs[0], hash)
	require.NoError(t, os.WriteFile(firstPath, []byte("first"), 0o600))
	require.NoError(t, os.Mkdir(filepath.Join(dirs[1], hash[:2]), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dirs[1], hash[:2], hash+".unity3d"), []byte("second"), 0o600))

	for _, expected := range []string{"first", "second"} {
		file, err := h.OpenResourceByHash(hash)
		require.NoError(t, err)
		content, err := io.ReadAll(file)
		file.Close()
		require.NoError(t, err)
		require.Equal(t, expected, string(content))
		if expected == "first" {
			require.NoError(t, os.Remove(firstPath))
		}
	}

	for _, missing := range []string{"", "a", "missing"} {
		file, err := h.OpenResourceByHash(missing)
		require.ErrorIs(t, err, os.ErrNotExist)
		require.Nil(t, file)
	}
	require.NoError(t, h.assets[0].Close())
	file, err := h.OpenResourceByHash("missing")
	require.Nil(t, file)
	require.ErrorIs(t, err, os.ErrClosed)
	require.ErrorIs(t, err, os.ErrNotExist)
	joined, ok := err.(interface{ Unwrap() []error })
	require.True(t, ok)
	require.Len(t, joined.Unwrap(), 4)

	file, err = (&Handler{}).OpenResourceByHash(hash)
	require.ErrorIs(t, err, os.ErrNotExist)
	require.Nil(t, file)
}
