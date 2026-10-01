package flows

import (
	"bytes"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

const Limit = 256

type Summary struct {
	ID        uint64    `json:"id"`
	Time      time.Time `json:"time"`
	Method    string    `json:"method"`
	URL       string    `json:"url"`
	Status    int       `json:"status"`
	ProtoType string    `json:"protoType"`
}

type Record struct {
	Summary
	RequestHeaders  http.Header `json:"requestHeaders"`
	RequestBody     []byte      `json:"requestBody"`
	ResponseHeaders http.Header `json:"responseHeaders"`
	ResponseBody    []byte      `json:"responseBody"`
}

type Recorder struct {
	mu          sync.Mutex
	nextID      uint64
	records     []Record
	subscribers map[chan struct{}]struct{}
}

func (rec *Recorder) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		record := Record{
			Summary:        Summary{Time: time.Now(), Method: r.Method, URL: r.URL.RequestURI()},
			RequestHeaders: r.Header.Clone(),
		}
		var responseBody bytes.Buffer
		writer := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		writer.Tee(&responseBody)
		var err error
		if r.Body != nil {
			record.RequestBody, err = io.ReadAll(r.Body)
			r.Body.Close()
			r.Body = io.NopCloser(bytes.NewReader(record.RequestBody))
		}
		if err != nil {
			http.Error(writer, "Failed to read request body", http.StatusBadRequest)
		} else {
			next.ServeHTTP(writer, r)
		}
		record.Status = writer.Status()
		if record.Status == 0 {
			record.Status = http.StatusOK
		}
		record.ResponseHeaders = writer.Header().Clone()
		record.ProtoType = record.ResponseHeaders.Get("proto-type")
		record.ResponseBody = responseBody.Bytes()
		rec.add(record)
	})
}

func (rec *Recorder) add(record Record) {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	rec.nextID++
	record.ID = rec.nextID
	if len(rec.records) == Limit {
		copy(rec.records, rec.records[1:])
		rec.records[Limit-1] = record
	} else {
		rec.records = append(rec.records, record)
	}
	// Coalesce notifications so a slow viewer never delays a game request.
	for subscriber := range rec.subscribers {
		select {
		case subscriber <- struct{}{}:
		default:
		}
	}
}

func (rec *Recorder) List() []Summary {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	result := make([]Summary, len(rec.records))
	for i, record := range rec.records {
		result[i] = record.Summary
	}
	return result
}

func (rec *Recorder) Get(id uint64) (Record, bool) {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	for _, record := range rec.records {
		if record.ID == id {
			return record, true
		}
	}
	return Record{}, false
}

func (rec *Recorder) Subscribe() (<-chan struct{}, func()) {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	if rec.subscribers == nil {
		rec.subscribers = make(map[chan struct{}]struct{})
	}
	updates := make(chan struct{}, 1)
	rec.subscribers[updates] = struct{}{}
	return updates, func() {
		rec.mu.Lock()
		defer rec.mu.Unlock()
		delete(rec.subscribers, updates)
	}
}
