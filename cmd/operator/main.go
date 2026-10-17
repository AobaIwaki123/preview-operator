package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type OperatorConfig struct {
	Port         string
	TargetURL    string
	TargetNS     string
	TargetDeploy string
	IdleTimeout  time.Duration
}

type Controller struct {
	k8sClient   *kubernetes.Clientset
	cfg         OperatorConfig
	proxy       *httputil.ReverseProxy
	mu          sync.Mutex
	lastAccess  time.Time
	isWakingUp  bool
}

const waitingHTML = `<!DOCTYPE html>
<html lang="ja">
<head>
  <meta charset="UTF-8">
  <meta http-equiv="refresh" content="3">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Waking up Environment...</title>
  <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-slate-900 text-slate-100 min-h-screen flex items-center justify-center p-6">
  <div class="max-w-md w-full bg-slate-800/90 border border-slate-700 rounded-2xl p-8 shadow-2xl text-center space-y-6">
    <div class="flex justify-center">
      <div class="relative w-16 h-16">
        <div class="absolute inset-0 rounded-full border-4 border-indigo-500/20"></div>
        <div class="absolute inset-0 rounded-full border-4 border-indigo-400 border-t-transparent animate-spin"></div>
      </div>
    </div>
    <div class="space-y-2">
      <h1 class="text-xl font-bold text-white">Preview環境を起動中...</h1>
      <p class="text-xs text-slate-400">アイドル状態からPodを立ち上げています（約3秒で完了）</p>
    </div>
    <div class="p-3 bg-slate-900/60 rounded-xl border border-slate-700/40 text-xs text-slate-400 font-mono">
      Auto-refreshing in 3s...
    </div>
  </div>
</body>
</html>`

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func main() {
	cfg := OperatorConfig{
		Port:         getEnv("PORT", "8080"),
		TargetURL:    getEnv("TARGET_URL", "http://sample-app.preview-sample.svc.cluster.local:8080"),
		TargetNS:     getEnv("TARGET_NS", "preview-sample"),
		TargetDeploy: getEnv("TARGET_DEPLOY", "sample-app"),
		IdleTimeout:  5 * time.Minute,
	}

	targetURL, err := url.Parse(cfg.TargetURL)
	if err != nil {
		log.Fatalf("invalid target url: %v", err)
	}

	// In-cluster Kubernetes client
	k8sConfig, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("failed to get in-cluster k8s config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		log.Fatalf("failed to create k8s client: %v", err)
	}

	ctrl := &Controller{
		k8sClient:  clientset,
		cfg:        cfg,
		proxy:      httputil.NewSingleHostReverseProxy(targetURL),
		lastAccess: time.Now(),
	}

	// Sleeper goroutine (Check every 30s)
	go ctrl.runSleeperLoop()

	http.HandleFunc("/", ctrl.handleRequest)
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	})

	log.Printf("preview-operator started on port :%s (target: %s, idle timeout: %v)", cfg.Port, cfg.TargetURL, cfg.IdleTimeout)
	if err := http.ListenAndServe(":"+cfg.Port, nil); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func (c *Controller) handleRequest(w http.ResponseWriter, r *http.Request) {
	c.mu.Lock()
	c.lastAccess = time.Now()
	c.mu.Unlock()

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Check Deployment scale
	scale, err := c.k8sClient.AppsV1().Deployments(c.cfg.TargetNS).GetScale(ctx, c.cfg.TargetDeploy, metav1.GetOptions{})
	if err != nil {
		log.Printf("error getting scale: %v", err)
		http.Error(w, "error checking environment status", http.StatusInternalServerError)
		return
	}

	// If replicas == 0, wake up and return waiting HTML
	if scale.Spec.Replicas == 0 {
		log.Printf("waking up deployment %s/%s (0 -> 1)", c.cfg.TargetNS, c.cfg.TargetDeploy)
		scale.Spec.Replicas = 1
		_, err := c.k8sClient.AppsV1().Deployments(c.cfg.TargetNS).UpdateScale(ctx, c.cfg.TargetDeploy, scale, metav1.UpdateOptions{})
		if err != nil {
			log.Printf("failed to scale up: %v", err)
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, waitingHTML)
		return
	}

	// If ready, forward to app
	c.proxy.ServeHTTP(w, r)
}

func (c *Controller) runSleeperLoop() {
	ticker := time.NewTicker(30 * time.Second)
	for range ticker.C {
		c.mu.Lock()
		idleDuration := time.Since(c.lastAccess)
		c.mu.Unlock()

		if idleDuration > c.cfg.IdleTimeout {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			scale, err := c.k8sClient.AppsV1().Deployments(c.cfg.TargetNS).GetScale(ctx, c.cfg.TargetDeploy, metav1.GetOptions{})
			if err == nil && scale.Spec.Replicas > 0 {
				log.Printf("idle timeout reached (%v > %v). Scaling down %s/%s to 0", idleDuration, c.cfg.IdleTimeout, c.cfg.TargetNS, c.cfg.TargetDeploy)
				scale.Spec.Replicas = 0
				_, _ = c.k8sClient.AppsV1().Deployments(c.cfg.TargetNS).UpdateScale(ctx, c.cfg.TargetDeploy, scale, metav1.UpdateOptions{})
			}
			cancel()
		}
	}
}
