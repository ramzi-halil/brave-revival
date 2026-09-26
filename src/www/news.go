package www

import (
	"net/http"
)

func news(w http.ResponseWriter, _ *http.Request) {
	w.Write([]byte("<h1>Hello</h1>"))
}
