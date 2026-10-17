package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"
)

type PageData struct {
	AppName     string
	PRNumber    string
	CommitSHA   string
	Hostname    string
	CurrentTime string
}

const htmlTemplate = `<!DOCTYPE html>
<html lang="ja">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{ .AppName }} - Preview Environment</title>
  <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-slate-900 text-slate-100 min-h-screen flex items-center justify-center p-6">
  <div class="max-w-md w-full bg-slate-800/80 backdrop-blur border border-slate-700 rounded-2xl p-8 shadow-2xl space-y-6">
    <div class="flex items-center justify-between border-b border-slate-700/60 pb-4">
      <div class="flex items-center space-x-3">
        <span class="w-3 h-3 bg-emerald-400 rounded-full animate-ping"></span>
        <h1 class="text-xl font-bold tracking-tight text-white">{{ .AppName }}</h1>
      </div>
      <span class="px-2.5 py-1 text-xs font-semibold bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 rounded-full">
        Ready
      </span>
    </div>

    <div class="space-y-4 text-sm">
      <div class="flex justify-between items-center p-3 bg-slate-900/60 rounded-xl border border-slate-700/40">
        <span class="text-slate-400">PR Number</span>
        <span class="font-mono font-bold text-indigo-400">#{{ .PRNumber }}</span>
      </div>

      <div class="flex justify-between items-center p-3 bg-slate-900/60 rounded-xl border border-slate-700/40">
        <span class="text-slate-400">Commit SHA</span>
        <span class="font-mono text-xs text-amber-300">{{ .CommitSHA }}</span>
      </div>

      <div class="flex justify-between items-center p-3 bg-slate-900/60 rounded-xl border border-slate-700/40">
        <span class="text-slate-400">Pod Hostname</span>
        <span class="font-mono text-xs text-slate-300 truncate max-w-[180px]">{{ .Hostname }}</span>
      </div>

      <div class="flex justify-between items-center p-3 bg-slate-900/60 rounded-xl border border-slate-700/40">
        <span class="text-slate-400">Server Time</span>
        <span class="text-xs text-slate-400">{{ .CurrentTime }}</span>
      </div>
    </div>

    <div class="pt-4 border-t border-slate-700/60 text-center">
      <p class="text-xs text-slate-500">
        Powered by <span class="text-indigo-400 font-medium">k0s</span> & <span class="text-amber-400 font-medium">Cloudflare Tunnel</span>
      </p>
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
	port := getEnv("PORT", "8080")
	appName := getEnv("APP_NAME", "Sample Preview App")
	prNumber := getEnv("PR_NUMBER", "42")
	commitSHA := getEnv("COMMIT_SHA", "dev-local-sha")

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown-host"
	}

	tmpl, err := template.New("index").Parse(htmlTemplate)
	if err != nil {
		log.Fatalf("failed to parse template: %v", err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		data := PageData{
			AppName:     appName,
			PRNumber:    prNumber,
			CommitSHA:   commitSHA,
			Hostname:    hostname,
			CurrentTime: time.Now().Format("2006-01-02 15:04:05 JST"),
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.Execute(w, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	})

	log.Printf("Starting server on port :%s ...", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
