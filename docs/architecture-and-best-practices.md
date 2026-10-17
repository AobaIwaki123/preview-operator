# アーキテクチャ設計とKubernetesベストプラクティス

本ドキュメントは、`preview-operator` における内部構造・責務分離の設計思想と、Kubernetesコントローラー開発における標準的なベストプラクティスおよび参照リソースをまとめた設計メモです。

---

## 1. パッケージ構成と責務分離 (Clean Architecture)

コード肥大化を防ぎ、保守性とテスト容易性を高めるため、以下のレイヤード構造を採用しています。

| パッケージ | ファイル | 主な責務 | 依存関係 |
|---|---|---|---|
| `cmd/operator` | `main.go` | アプリケーションのエントリポイント。設定読み込み、各コンポーネントの初期化と依存注入、グレースフルシャットダウン管理。 | `pkg/*` |
| `pkg/config` | `config.go` | 環境変数（ポート、対象URL、Namespace、タイムアウト値）のパースと構造体提供。 | 標準ライブラリのみ |
| `pkg/k8s` | `client.go`, `scaler.go` | Kubernetes APIとの通信専用レイヤー。InCluster/Kubeconfigの切り替え、Deploymentのスケール取得（`GetReplicas`）および更新（`ScaleTo`）。 | `k8s.io/client-go` |
| `pkg/proxy` | `handler.go`, `template.go` | 受付リバースプロキシ。トラフィックを傍受し、未起動時は待合室HTML返却＆スケールアップ要求、起動済みなら本来のバックエンドへ透過プロキシ。 | `pkg/k8s`, `net/http` |
| `pkg/controller` | `sleeper.go` | アイドル監視タイマー。アクセス時刻をスレッドセーフに管理し、一定時間（5分）無アクセスの環境を自動スリープ（`replicas=0`）。 | `pkg/k8s` |

---

## 2. コントローラー設計のトレードオフ：Lightweight vs Kubebuilder

Kubernetesのカスタムコントローラー開発には大きく分けて2つのアプローチがあります：

1. **Kubebuilder / Operator-SDK（フルスタックフレームワーク）**:
   - **特徴**: CRDのコード生成、Webhook、Leader Election、Metricsエンドポイント、Informerキャッシュが自動生成される。
   - **適合**: 多数のリソースを管理する大規模・商用Operator。
2. **Lightweight Controller（本プロジェクトの採用方式）**:
   - **特徴**: `client-go` の低レベルAPIを直接使い、標準GoのHTTPサーバーおよびゴルーチンとして最小限で実装。
   - **採用理由**:
     - 単一DeploymentのScale操作に限定されており、CRDの生成コストや数千行のボイラープレートが不要（YAGNI）。
     - コードの全体像が数ファイル・数百行で把握でき、学習コストとデバッグ容易性が最大化される。

---

## 3. 権威あるベストプラクティス・参照リソース

今後、本格的なCRD定義やOperator拡張を行う際に参照すべき業界標準リソースです。

- **[The Kubebuilder Book](https://book.kubebuilder.io/)** (Kubernetes SIGs 公式):
  Operatorおよびコントローラー開発の公式バイブル。Reconciliation Loopのライフサイクル、Informerキャッシュの仕組みが網羅されています。
- **[Programming Kubernetes](https://www.oreilly.com/library/view/programming-kubernetes/9781492047100/)** (Michael Hausenblas, Stefan Schimanski 著 / O'Reilly):
  KubernetesのAPI機械構造、client-goの拡張、カスタムコントローラーのプログラミングに関する決定版。
- **[Kubernetes Configuration Best Practices](https://kubernetes.io/docs/concepts/configuration/overview/)** (Kubernetes公式ドキュメント):
  マニフェスト定義、リソースリクエスト/リミット、シグナル処理に関する公式ガイドライン。
- **[Production Best Practices Checklist](https://learnk8s.io/production-best-practices)** (Learnk8s):
  本番環境における可用性、セキュリティ、オブザーバビリティのアンチパターン防止チェックリスト。
