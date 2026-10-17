---
schema_version: 1
id: "iss-2610140813542607"
slug: "internal-core-history-store-go-22-and"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "fixing iss-2610090740139804 (ideate private/public stores)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/history/store.go"
remedy: "Replace the rootSHARe copies in internal/core/history/store.go and internal/core/lifeboat/voyage.go with gitutil.IsFullSHA, keeping each package's existing tests green; leave lint's receiptShaRe and report's senderKeyRe, which key different things, unless their owners agree they are the same check."
---

internal/core/history/store.go:22 and internal/core/lifeboat/voyage.go:25 each declare their own rootSHARe (^(?:[0-9a-f]{40}|[0-9a-f]{64})$) to check a store key is a repository root commit, while gitutil.IsFullSHA (internal/gitutil/repo.go) already carries the same pattern for the same purpose and is what the docfidelity, lab, implement and ideate stores call. The two copies can drift from it (one-canonical-primitive). Found while fixing iss-2610090740139804, whose first draft added a third copy before review pointed at IsFullSHA.
