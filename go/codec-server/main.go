package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"money-transfer-worker/encryption"

	"go.temporal.io/sdk/converter"
)

func main() {
	port := getEnv("CODEC_SERVER_PORT", "8081")

	codec := &encryption.Codec{KeyID: "test"}
	handler := converter.NewPayloadCodecHTTPHandler(codec)

	mux := http.NewServeMux()
	mux.Handle("/", withCORS(handler))

	addr := fmt.Sprintf(":%s", port)
	log.Printf("Codec server listening on %s", addr)
	log.Printf("Configure Temporal UI with codec endpoint: http://localhost:%s", port)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}

		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Namespace, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
