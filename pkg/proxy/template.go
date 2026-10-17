package proxy

const WaitingHTML = `<!DOCTYPE html>
<html lang="ja">
<head>
  <meta charset="UTF-8">
  <meta http-equiv="refresh" content="3">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Preview環境を起動中...</title>
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
      <h1 class="text-xl font-bold text-white tracking-tight">Preview環境を起動中...</h1>
      <p class="text-xs text-slate-400">アイドル状態からPodを立ち上げています（約3秒で完了）</p>
    </div>
    <div class="p-3 bg-slate-900/60 rounded-xl border border-slate-700/40 text-xs text-indigo-400 font-mono">
      3秒後に自動リロードします...
    </div>
  </div>
</body>
</html>`
