package main

import (
    "log/slog"
    "net"
    "net/http"
    "os"

    "github.com/go-chi/chi/v5"
)

func main() {
	host := os.Getenv("APP_HOST")
	port := os.Getenv("APP_PORT")

	if port == "" {
		port = "8080"
	}
	addr := net.JoinHostPort(host, port)

	r := chi.NewRouter()
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte("ok"))
    })

    slog.Info("inventory service запущен", slog.String("адрес", addr))
    if err := http.ListenAndServe(addr, r); err != nil {
        slog.Error("не удалось запустить HTTP-сервер", slog.String("адрес", addr), slog.Any("ошибка", err))
        os.Exit(1)
    }
}