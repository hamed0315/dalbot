package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"dalbot/api"
	"dalbot/logger"
	"dalbot/services"
)

// StartWebhookServer starts an HTTP server and processes incoming updates immediately,
// bypassing the msgQueue to reduce latency.
func StartWebhookServer(listenAddr string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Error.Printf("Webhook read error: %v", err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		var upd api.Update
		if err := json.Unmarshal(body, &upd); err != nil {
			logger.Error.Printf("Webhook unmarshal error: %v", err)
			http.Error(w, "Bad JSON", http.StatusBadRequest)
			return
		}

		// پردازش مستقیم آپدیت، بدون عبور از صف
		services.AcquireToken()
		ProcessUpdate(&upd)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	logger.Info.Printf("Webhook server listening on %s", listenAddr)
	if err := http.ListenAndServe(listenAddr, mux); err != nil {
		logger.Error.Fatalf("Webhook server fatal: %v", err)
	}
}
