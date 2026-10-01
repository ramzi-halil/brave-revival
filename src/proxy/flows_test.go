package proxy

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"example.com/brave-revival/src/config"
	"example.com/brave-revival/src/flows"
	"example.com/brave-revival/src/proto/proto"
	"example.com/brave-revival/src/www"
	"github.com/stretchr/testify/require"
	pb "google.golang.org/protobuf/proto"
)

func TestFlowViewer(t *testing.T) {
	h := newHandler(&config.Config{}, tls.Certificate{}, &www.Handler{})
	h.flows = &flows.Recorder{}
	body, err := pb.Marshal(&proto.AccountAuthorize{Token: "example"})
	require.NoError(t, err)
	record := h.flows.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("proto-type", "Proto.AccountAuthorize")
		w.Write(body)
	}))

	server := httptest.NewServer(h)
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/forms/flows", nil)
	require.NoError(t, err)
	response, err := server.Client().Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	require.Equal(t, "text/event-stream", response.Header.Get("Content-Type"))
	scanner := bufio.NewScanner(response.Body)
	readEvent := func() []flows.Summary {
		t.Helper()
		for scanner.Scan() {
			if data, ok := strings.CutPrefix(scanner.Text(), "data: "); ok {
				var summaries []flows.Summary
				require.NoError(t, json.Unmarshal([]byte(data), &summaries))
				return summaries
			}
		}
		t.Fatalf("event stream ended: %v", scanner.Err())
		return nil
	}
	require.Empty(t, readEvent())
	record.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/v1_43_274/account/authorize?x=1", strings.NewReader("session=test")))
	summaries := readEvent()
	require.Len(t, summaries, 1)
	require.Equal(t, "/v1_43_274/account/authorize?x=1", summaries[0].URL)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/forms/flows/1", nil))
	require.Equal(t, http.StatusOK, w.Code)
	var detail struct {
		flows.Record
		RequestText  string
		ResponseText string
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &detail))
	require.Equal(t, "session=test", detail.RequestText)
	require.JSONEq(t, `{"token":"example"}`, detail.ResponseText)
	require.Equal(t, body, detail.ResponseBody)

	for url, status := range map[string]int{"/flows": 200, "/forms/flows/999": 404, "/forms/flows/invalid": 400} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, url, nil))
		require.Equal(t, status, w.Code)
		if url == "/flows" {
			require.Contains(t, w.Body.String(), "<style>")
			require.Contains(t, w.Body.String(), "<script>")
		}
	}
	require.Len(t, h.flows.List(), 1)
}

func TestFlowBodyDisplay(t *testing.T) {
	require.Equal(t, "Base64:\nAP8=", responseText(flows.Record{ResponseBody: []byte{0, 0xff}}))
	require.Equal(t, "plain text", responseText(flows.Record{ResponseBody: []byte("plain text")}))
	require.JSONEq(t, `{}`, responseText(flows.Record{Summary: flows.Summary{ProtoType: "Proto.Empty"}}))
	require.Equal(t, "Base64:\n/w==", responseText(flows.Record{Summary: flows.Summary{ProtoType: "Proto.Empty"}, ResponseBody: []byte{0xff}}))
}
