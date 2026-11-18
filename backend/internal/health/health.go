package health

import (
	"atlas-financial-core/internal/response"
	"fmt"
	"net/http"
)

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		response.ErrorWriter(w, 405, "METHOD_NOT_ALLOWED", "Method is not allowed")
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK")
}

func HealthRoute(mux *http.ServeMux) {
	mux.HandleFunc("/health", HealthHandler)
}
