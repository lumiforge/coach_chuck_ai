package a2a

import (
	"io"
	"log"
	"net/http"
	"strings"
)

func withRequestDump(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var bodyBytes []byte
		if r.Body != nil {
			bodyBytes, _ = io.ReadAll(r.Body)
			r.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
		}

		log.Printf("HTTP %s %s from %s", r.Method, r.URL.String(), r.RemoteAddr)
		log.Printf("Headers: %v", r.Header)
		if len(bodyBytes) > 0 {
			log.Printf("Body: %s", string(bodyBytes))
		} else {
			log.Printf("Body: <empty>")
		}

		next.ServeHTTP(w, r)
	})
}

func withEndpointLog(name string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf(
			"%s: method=%s path=%s content-type=%s accept=%s",
			name,
			r.Method,
			r.URL.Path,
			r.Header.Get("Content-Type"),
			r.Header.Get("Accept"),
		)
		next.ServeHTTP(w, r)
	})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}

		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
