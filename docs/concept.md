# Concept & Motivation: Why preview-operator?

In Kubernetes-based development workflows, providing isolated, on-demand preview environments for every Pull Request is one of the most effective ways to accelerate code reviews. However, in resource-constrained environments (such as bare-metal clusters, homelabs, or cost-sensitive development clusters), running dozens of active preview deployments simultaneously causes severe CPU and memory contention.

---

## Evolution: As-Is vs To-Be

### As-Is: CI-Driven Manual Pipelines (Heavy & Resource-Intensive)

In traditional setups, CI runs custom bash scripts or automation to generate manifests and deploy preview pods that run 24/7 until manually cleaned up.

```mermaid
%%{init: {"flowchart": {"curve": "basis"}}}%%
flowchart TD
    subgraph GitHub ["GitHub"]
        PR["Pull Request #123"]
        CI["GitHub Actions"]
        GitRepo["Git Repository"]
    end

    subgraph Cluster ["Kubernetes Cluster"]
        ArgoCD["Argo CD"]
        Pod["Preview Pod (24/7 Running / Memory Hog)"]
        Ingress["Ingress"]
    end

    PR -->|"Trigger"| CI
    CI -->|"Generate Manifests & Commit"| GitRepo
    GitRepo -->|"Sync"| ArgoCD
    ArgoCD -->|"Deploy"| Pod
    Ingress -->|"Direct Routing"| Pod

    style GitRepo fill:#ffe6e6,stroke:#ff4d4d
    style Pod fill:#ffe6e6,stroke:#ff4d4d
```

**Pain Points:**
- **Resource Exhaustion**: Pods run continuously even when reviewers are away or sleeping.
- **Git Repo Pollution**: Generating dynamic manifests directly in Git repos clutters commit history.
- **Cost**: Unnecessary cloud compute bills or OOMKills in local bare-metal clusters.

---

### To-Be: Operator-Driven Scale-to-Zero & Instant Wakeup

With `preview-operator`, preview pods consume cluster resources **only when being actively viewed**.

```mermaid
%%{init: {"flowchart": {"curve": "basis"}}}%%
flowchart TD
    subgraph Reviewer ["Reviewer"]
        Browser["Web Browser"]
    end

    subgraph Cluster ["Kubernetes Cluster"]
        Ingress["Ingress"]
        Operator["preview-operator (Waiting Room Proxy + Sleeper)"]
        Pod["Preview Pod (Replicas: 0 or 1)"]
    end

    Browser -->|"HTTP Request"| Ingress
    Ingress -->|"Forward"| Operator
    Operator -->|"Pod stopped? Scale 0 -> 1 & Show Waiting Room"| Pod
    Operator -->|"Pod ready? Reverse proxy traffic"| Pod
    Operator -->|"Idle for N minutes? Scale 1 -> 0"| Pod

    style Operator fill:#e6ffe6,stroke:#33cc33
    style Pod fill:#fff2e6,stroke:#ff9933
```

**Key Advantages:**
1. **Zero Footprint When Idle**: Scales deployments to 0 replicas after an idle period.
2. **Transparent User Experience**: Reviewers see a fast, auto-refreshing "Waking up preview..." page rather than a 502/503 Bad Gateway error.
3. **No Complex Infrastructure**: No Service Mesh (Istio/Linkerd) or Knative required. Just a single Go container deployed alongside your app.
