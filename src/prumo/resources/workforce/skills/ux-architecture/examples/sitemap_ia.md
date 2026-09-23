# Enterprise Platform Information Architecture & Sitemap Model

## 1. Hierarchy Overview & Metrics
- **Platform**: Cloud Developer Platform
- **Top-Level Categories**: 5 items (Dashboard, Repositories, Deployments, Monitoring, Settings)
- **Maximum Tree Depth**: 3 levels
- **Orphaned Views**: 0

---

## 2. Mermaid Structural Sitemap

```mermaid
graph TD
    Root([Global Dashboard])
    
    %% Tier 1 Modules
    Root --> Repos[1.0 Repositories]
    Root --> Deploys[2.0 Deployments]
    Root --> Monitor[3.0 Monitoring]
    Root --> Access[4.0 Identity & Access]
    Root --> Settings[5.0 Organization Settings]
    
    %% Tier 2 Sub-Sections
    Repos --> RepoList[1.1 All Repositories]
    Repos --> RepoDetail[1.2 Repository Inspector]
    
    RepoDetail --> CodeView[1.2.1 Source Code Tree]
    RepoDetail --> PRView[1.2.2 Pull Requests]
    RepoDetail --> CommitView[1.2.3 Commit History]
    
    Deploys --> Pipelines[2.1 CI/CD Pipelines]
    Deploys --> Releases[2.2 Production Releases]
    
    Monitor --> Metrics[3.1 Telemetry & Metrics]
    Monitor --> Alerts[3.2 Alert Rules & Incidents]
    
    Access --> Users[4.1 Team Members]
    Access --> Roles[4.2 RBAC Permission Policies]
```

---

## 3. Wayfinding & Breadcrumb Specifications

When navigating to a specific Pull Request (`PR #142`):
```text
Home / Repositories / prumo-core / Pull Requests / #142: Fix Memory Arena
```

HTML Implementation Contract:
```html
<nav aria-label="Breadcrumb">
  <ol class="flex items-center space-x-2 text-sm text-gray-600">
    <li><a href="/" class="hover:underline">Home</a></li>
    <li aria-hidden="true">/</li>
    <li><a href="/repos" class="hover:underline">Repositories</a></li>
    <li aria-hidden="true">/</li>
    <li><a href="/repos/prumo-core" class="hover:underline">prumo-core</a></li>
    <li aria-hidden="true">/</li>
    <li><a href="/repos/prumo-core/pulls" class="hover:underline">Pull Requests</a></li>
    <li aria-hidden="true">/</li>
    <li><span aria-current="page" class="font-semibold text-gray-900">#142: Fix Memory Arena</span></li>
  </ol>
</nav>
```
