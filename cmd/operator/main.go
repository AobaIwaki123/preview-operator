package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AobaIwaki123/preview-operator/pkg/config"
	"github.com/AobaIwaki123/preview-operator/pkg/controller"
	"github.com/AobaIwaki123/preview-operator/pkg/k8s"
	"github.com/AobaIwaki123/preview-operator/pkg/proxy"
)

func main() {
	// 1. 設定の読み込み
	cfg := config.Load()

	// 2. Kubernetes クライアントの初期化
	k8sClient, err := k8s.NewClient()
	if err != nil {
		log.Fatalf("failed to initialize k8s client: %v", err)
	}

	// 3. Deployment スケーラーの初期化
	scaler := k8s.NewScaler(k8sClient, cfg.TargetNS, cfg.TargetDeploy)

	// 4. アイドル監視コントローラー（Sleeper）の初期化
	sleeper := controller.NewSleeper(scaler, cfg.IdleTimeout)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// バックグラウンドでスリーパーを起動
	go sleeper.Start(ctx)

	// 5. 待合室プロキシの初期化（アクセス時に sleeper.RecordAccess を呼ぶ）
	proxyHandler, err := proxy.NewHandler(scaler, cfg.TargetURL, sleeper.RecordAccess)
	if err != nil {
		log.Fatalf("failed to initialize proxy handler: %v", err)
	}

	// 6. HTTP サーバーの起動
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: proxyHandler,
	}

	go func() {
		log.Printf("preview-operator listening on :%s (target: %s, idleTimeout: %v)", cfg.Port, cfg.TargetURL, cfg.IdleTimeout)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// グレースフルシャットダウン待機
	<-ctx.Done()
	log.Println("Shutting down preview-operator gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
	log.Println("preview-operator stopped.")
}
