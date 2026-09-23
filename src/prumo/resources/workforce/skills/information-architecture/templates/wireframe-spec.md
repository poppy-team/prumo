# Information Architecture Specification — Merchant Settings

## Inventory
- **Top-level domains**: Profile, Team, Billing, Integrations, Security, Notifications
- **Critical tasks**: find payout destination, rotate API key, invite employee, download invoice
- **Current issue**: Billing requires three clicks; API key rotation requires five

## Navigation Tree
```text
Settings
├── Profile
│   ├── Personal details
│   └── Support contacts
├── Team
│   ├── Members
│   └── Roles
├── Billing
│   ├── Payment methods
│   ├── Payout destination
│   └── Invoice history
├── Integrations
│   └── API keys
├── Security
│   ├── Sign-in methods
│   └── Active sessions
└── Notifications
    ├── Email
    └── Webhooks
```

Primary navigation depth is two; leaf screens do not exceed depth three. Each critical task has a direct settings path and a global search path.

## Label Decisions
- Use **Payout destination** instead of `Settlement configuration`; the former matches support and product vocabulary.
- Use **API keys** instead of `Developer tokens`; five of eight card-sort participants selected the former.
- Keep **Team** and **Security** at the same level to avoid implying one is subordinate to the other.

## Wayfinding Acceptance
- Tree-test direct success is at least 85% for each critical task.
- No orphan, dead-end, duplicate label, or unlabeled icon remains.
- Legal and security destinations remain within two navigation actions.
