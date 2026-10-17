package proxy

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/AobaIwaki123/preview-operator/pkg/k8s"
)

// Handler handles incoming HTTP requests, wakes up idle deployments, or proxies to target.
type Handler struct {
	scaler        *k8s.Scaler
	proxy         *httputil.ReverseProxy
	onAccess      func()
}

// NewHandler creates a new proxy Handler.
func NewHandler(scaler *k8s.Scaler, targetURLStr string, onAccess func()) (*Handler, error) {
	targetURL, err := url.Parse(targetURLStr)
	if err != nil {
		return nil, fmt.Errorf("invalid target URL: %w", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("[Proxy] Target backend not yet reachable (%v). Returning waiting room...", err)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, WaitingHTML)
	}

	return &Handler{
		scaler:   scaler,
		proxy:    proxy,
		onAccess: onAccess,
	}, nil
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// ヘルスチェックはそのまま返す
	if r.URL.Path == "/healthz" {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
		return
	}

	// アクセス記録を更新
	if h.onAccess != nil {
		h.onAccess()
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Deployment の状態確認（Desired および Ready レプリカ数）
	status, err := h.scaler.GetStatus(ctx)
	if err != nil {
		log.Printf("[Proxy] Error checking deployment status: %v", err)
		http.Error(w, "failed to check environment status", http.StatusInternalServerError)
		return
	}

	// 1. スリープ中（Desired == 0）なら起動指示を出して待合室HTMLを返す
	if status.DesiredReplicas == 0 {
		log.Printf("[Proxy] Target is sleeping (desired=0). Waking up to 1...")
		if err := h.scaler.ScaleTo(ctx, 1); err != nil {
			log.Printf("[Proxy] Error scaling up: %v", err)
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, WaitingHTML)
		return
	}

	// 2. 起動中（Desired > 0 だが Pod が Ready でない）なら待合室HTMLを返す
	if status.ReadyReplicas == 0 {
		log.Printf("[Proxy] Target is starting up (ready=0, desired=%d). Serving waiting room...", status.DesiredReplicas)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, WaitingHTML)
		return
	}

	// 3. Pod が Ready になっていれば本来のアプリケーションへリバースプロキシ
	h.proxy.ServeHTTP(w, r)
}
