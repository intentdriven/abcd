---
schema_version: 1
id: "iss-2609012037137250"
slug: "sibling-of-ghsa-865x-5m7q-qm79-cwe-59-found-on-the-sweep-and"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/lifecycle.go"
---

Sibling of GHSA-865x-5m7q-qm79 (CWE-59), found on the sweep and not fixed there: intent ensureRealDir (internal/core/intent/lifecycle.go) and spec ensureDir (internal/core/spec/store.go) refuse a symlinked LEAF directory and then os.MkdirAll the path, so a committed symlink at an ancestor (.abcd, .abcd/development, or intents/ and specs/ themselves) redirects the store create and every subsequent write — including the draft that capture promote mints through intent.CreateDraft — outside the checkout. Both functions carry a NOTE naming this as a low-severity follow-up under the trusted-worktree model; that model exists only in those comments and a research note, not in an ADR, and the same class is already ruled in scope for memory (ad2f1a8e), ahoy (81f81f67) and now the issue ledger. The fix is the per-segment Lstat walk from repoRoot that memory.memoryDir and the capture ledger use, one level at a time with os.Mkdir, with a test per store that symlinks .abcd/development out of tree and asserts a refusal and an empty target. Record-store containment as a structural rule is iss-2608301308367566; this record names the two create sites that rule would close.
