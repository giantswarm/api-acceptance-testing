package load

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func Test_ProduceLoad(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`OK`))
	}))
	defer server.Close()

	ProduceLoad(server.URL, 15*time.Second, 1_000_000_000)
}
