package flows

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMiddlewarePreservesBodies(t *testing.T) {
	var rec Recorder
	body := "value=" + strings.Repeat("abc", 10000)
	r := httptest.NewRequest(http.MethodPost, "/v1_43_274/test?x=a%2Fb&x=c", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Add("X-Test", "one")
	r.Header.Add("X-Test", "two")
	w := httptest.NewRecorder()
	rec.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		require.Equal(t, strings.TrimPrefix(body, "value="), r.PostForm.Get("value"))
		w.Header().Set("proto-type", "Test.Message")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte{0, 0xff})
		w.Write([]byte("payload"))
	})).ServeHTTP(w, r)

	record, ok := rec.Get(1)
	require.True(t, ok)
	require.Equal(t, http.MethodPost, record.Method)
	require.Equal(t, "/v1_43_274/test?x=a%2Fb&x=c", record.URL)
	require.Equal(t, body, string(record.RequestBody))
	require.Equal(t, []string{"one", "two"}, record.RequestHeaders.Values("X-Test"))
	require.Equal(t, http.StatusCreated, record.Status)
	require.Equal(t, "Test.Message", record.ProtoType)
	require.Equal(t, w.Body.Bytes(), record.ResponseBody)
	require.Equal(t, w.Result().Header, record.ResponseHeaders)
}

func TestRecorderConcurrentRetentionAndSubscriptions(t *testing.T) {
	var rec Recorder
	updates, unsubscribe := rec.Subscribe()
	otherUpdates, otherUnsubscribe := rec.Subscribe()
	defer otherUnsubscribe()
	handler := rec.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, r.URL.Path)
	}))
	const count = Limit + 40
	var wg sync.WaitGroup
	for i := range count {
		wg.Go(func() {
			handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, fmt.Sprintf("/test/%d", i), nil))
			rec.List()
			rec.Get(1)
		})
	}
	finished := make(chan struct{})
	go func() {
		wg.Wait()
		close(finished)
	}()
	select {
	case <-finished:
	case <-time.After(5 * time.Second):
		t.Fatal("requests blocked on slow subscribers")
	}
	summaries := rec.List()
	require.Len(t, summaries, Limit)
	for i, summary := range summaries {
		require.Equal(t, uint64(count-Limit+i+1), summary.ID)
		record, ok := rec.Get(summary.ID)
		require.True(t, ok)
		require.Equal(t, http.StatusOK, record.Status)
		require.Equal(t, record.URL, string(record.ResponseBody))
	}
	_, ok := rec.Get(1)
	require.False(t, ok)
	<-updates
	<-otherUpdates
	unsubscribe()
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/last", nil))
	select {
	case <-updates:
		t.Fatal("received an update after unsubscribing")
	default:
	}
	select {
	case <-otherUpdates:
	default:
		t.Fatal("the remaining subscriber missed an update")
	}
}
