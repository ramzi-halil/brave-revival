package www

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"example.com/brave-revival/src/config"
	"example.com/brave-revival/src/proto/pmaster"
	"github.com/stretchr/testify/require"
)

func TestVersionedFlows(t *testing.T) {
	dbDir := t.TempDir()
	fields := (&pmaster.All{}).ProtoReflect().Descriptor().Fields()
	for i := range fields.Len() {
		field := fields.Get(i)
		rowFields := field.Message().Fields()
		columns := make([]string, rowFields.Len())
		for j := range rowFields.Len() {
			columns[j] = string(rowFields.Get(j).Name())
		}
		content := strings.Join(columns, ",") + "\n"
		if field.Name() == "version" {
			content = "platform,application,resource,master\nandroid,1.44.274,6749,12810\n"
		}
		require.NoError(t, os.WriteFile(filepath.Join(dbDir, string(field.Name())+".csv"), []byte(content), 0o600))
	}
	require.NoError(t, os.WriteFile(filepath.Join(dbDir, "resources.csv"), []byte("id,ios,android,ios_size,android_size\n"), 0o600))
	h, err := NewHandler(&config.Config{DBDir: dbDir, AssetsDir: t.TempDir()})
	require.NoError(t, err)
	t.Cleanup(func() { h.assets.Close() })

	cases := []struct {
		method    string
		url       string
		body      string
		status    int
		protoType string
	}{
		{http.MethodPost, "/v1_43_274/account/certificate?reason=initial", "session=example", 200, "Proto.AccountCertificate"},
		{http.MethodGet, "/v1_44_274/etc", "", 200, "Proto.Etc"},
		{http.MethodPost, "/v1_44_274/unknown?x=a%2Fb", "unread=body", 404, "Pcommon.Error"},
		{http.MethodPut, "/v1_43_274/account/authorize", "", 404, "Pcommon.Error"},
		{http.MethodGet, "/v1_43_274", "", 404, "Pcommon.Error"},
	}
	for i, tc := range cases {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(tc.method, tc.url, strings.NewReader(tc.body)))
		require.Equal(t, tc.status, w.Code)
		record, ok := h.Flows().Get(uint64(i + 1))
		require.True(t, ok, tc.url)
		require.Equal(t, tc.method, record.Method)
		require.Equal(t, tc.url, record.URL)
		require.Equal(t, tc.body, string(record.RequestBody))
		require.Equal(t, tc.status, record.Status)
		require.Equal(t, tc.protoType, record.ProtoType)
		require.Equal(t, w.Body.Bytes(), record.ResponseBody)
		require.Equal(t, "12810", record.ResponseHeaders.Get("x-enish-app-version-master"))
	}
	for _, url := range []string{"/news/top/android", "/unknown", "/v1_45_274/etc"} {
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, url, nil))
	}
	require.Len(t, h.Flows().List(), len(cases))
}
