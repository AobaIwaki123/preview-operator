# ArgoCD ApplicationSet Integration Guide

`preview-operator` can be seamlessly integrated with ArgoCD's Pull Request generator (`ApplicationSet`) to enable full-lifecycle preview environments with automated scale-to-zero capabilities.

---

## Complete Workflow Architecture

```mermaid
%%{init: {"flowchart": {"curve": "basis"}}}%%
flowchart TD
    subgraph GitHub ["GitHub"]
        PR["Pull Request"]
        CI["GitHub Actions (Build & Push)"]
    end

    subgraph Registry ["Container Registry"]
        GHCR["GHCR (ghcr.io)"]
    end

    subgraph Cluster ["Kubernetes Cluster"]
        subgraph ArgoCD ["Argo CD"]
            AppSet["ApplicationSet (PR Generator)"]
        end

        subgraph Operator ["preview-operator"]
            Proxy["Wakeup Proxy"]
            Sleeper["Sleeper Controller"]
        end

        App["Preview App (Deployment)"]
        Ingress["Ingress (e.g. Cloudflare)"]
    end

    PR -->|"Trigger"| CI
    CI -->|"Push Image"| GHCR
    PR -.->|"Poll PRs"| AppSet
    AppSet -->|"Deploy / Prune"| App

    Ingress -->|"HTTP Traffic"| Proxy
    Proxy -->|"Scale 0 -> 1"| App
    Proxy -->|"Reverse Proxy"| App
    Sleeper -->|"Idle Scale 1 -> 0"| App
```

---

## ArgoCD ApplicationSet Manifest

Below is a production-ready `ApplicationSet` manifest targeting GitHub pull requests:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: preview-environments
  namespace: argocd
spec:
  generators:
    - pullRequest:
        github:
          owner: <GITHUB_OWNER>
          repo: <GITHUB_REPO>
          tokenRef:
            secretName: github-token
            key: token
  template:
    metadata:
      name: 'preview-pr-{{number}}'
    spec:
      project: default
      source:
        repoURL: 'https://github.com/<GITHUB_OWNER>/<GITHUB_REPO>'
        targetRevision: '{{head_sha}}'
        path: k8s/preview
        helm:
          parameters:
            - name: image.tag
              value: '{{head_sha}}'
            - name: ingress.host
              value: 'pr-{{number}}.preview.example.com'
      destination:
        server: https://kubernetes.default.svc
        namespace: 'pr-{{number}}'
      syncPolicy:
        automated:
          prune: true
          selfHeal: true
```

### Key Highlights
- **Dynamic Namespace Isolation**: Each PR gets its own namespace (`pr-{{number}}`), ensuring multi-tenant isolation.
- **Automated Lifecycle**: When the PR is closed or merged, ArgoCD automatically prunes the namespace, removing all related deployments and ingress resources.
- **Resource Protection**: Paired with `preview-operator`, idle preview environments sleep after inactivity, conserving cluster CPU and RAM.
