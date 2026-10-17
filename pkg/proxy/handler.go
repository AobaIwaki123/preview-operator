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

	return &Handler{
		scaler:   scaler,
		proxy:    httputil.NewSingleHostReverseProxy(targetURL),
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

	// レプリカ数の確認
	replicas, err := h.scaler.GetReplicas(ctx)
	if err != nil {
		log.Printf("[Proxy] Error checking deployment replicas: %v", err)
		http.Error(w, "failed to check environment status", http.StatusInternalServerError)
		return
	}

	// スリープ中（replicas == 0）なら起動指示を出して待合室HTMLを返す
	if replicas == 0 {
		log.Printf("[Proxy] Target is sleeping (replicas=0). Waking up...")
		if err := h.scaler.ScaleTo(ctx, 1); err != nil {
			log.Printf("[Proxy] Error scaling up: %v", err)
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, WaitingHTML)
		return
	}

	// 起動済みなら本来のアプリケーションへリバースプロキシ
	h.proxy.ServeHTTP(w, r)
}
