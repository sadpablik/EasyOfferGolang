package main

import (
	"bytes"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	_ "easyoffer/gateway/docs"

	httpSwagger "github.com/swaggo/http-swagger/v2"
)

func main() {
	authURL := strings.TrimRight(strings.TrimSpace(os.Getenv("AUTH_SERVICE_URL")), "/")

	if authURL == "" {
		log.Fatal("AUTH_SERVICE_URL is required")
	}

	port := strings.TrimSpace(os.Getenv("GATEWAY_PORT"))
	if port == "" {
		port = "8080"
	}

	g := &gateway{
		client:  &http.Client{Timeout: 5 * time.Second},
		authURL: authURL,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/auth/register", g.registerHandler)
	mux.HandleFunc("/api/v1/auth/login", g.loginHandler)
	mux.HandleFunc("/health", healthHandler)
	mux.Handle("/swagger/", httpSwagger.Handler(httpSwagger.URL("/swagger/doc.json")))

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("gateway starting on :%s", port)
	log.Fatal(server.ListenAndServe())
}

// registerHandler proxies registration requests to Auth Service.
// @Summary Register user
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "User registration"
// @Success 201 {object} RegisterResponse
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/auth/register [post]
func (g *gateway) registerHandler(w http.ResponseWriter, r *http.Request) {
	g.proxyPost(w, r, g.authURL+"/register")
}

// loginHandler proxies login requests to Auth Service.
// @Summary Login user
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "User login"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/auth/login [post]
func (g *gateway) loginHandler(w http.ResponseWriter, r *http.Request) {
	g.proxyPost(w, r, g.authURL+"/login")
}

// healthHandler returns service liveness status.
// @Summary Gateway health check
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (g *gateway) proxyPost(w http.ResponseWriter, r *http.Request, target string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	upstreamReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		http.Error(w, "failed to build upstream request", http.StatusInternalServerError)
		return
	}

	if ct := strings.TrimSpace(r.Header.Get("Content-Type")); ct != "" {
		upstreamReq.Header.Set("Content-Type", ct)
	} else {
		upstreamReq.Header.Set("Content-Type", "application/json")
	}

	resp, err := g.client.Do(upstreamReq)
	if err != nil {
		if errors.Is(err, io.EOF) {
			http.Error(w, "upstream closed connection", http.StatusBadGateway)
			return
		}
		http.Error(w, "upstream service unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "failed to read upstream response", http.StatusBadGateway)
		return
	}

	if ct := strings.TrimSpace(resp.Header.Get("Content-Type")); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(respBody)
}
