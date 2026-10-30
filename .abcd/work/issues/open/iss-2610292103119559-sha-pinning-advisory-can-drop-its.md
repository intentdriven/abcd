---
schema_version: 1
id: "iss-2610292103119559"
slug: "sha-pinning-advisory-can-drop-its"
severity: "minor"
category: "drift"
source: "impl-review"
found_during: "fidelity review rcp-6083de2cef52 of itd-2610290746077716"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/remote_baseline.go"
remedy: "Give the unpinned count in the SHA-pinning advisory whenever abcd read any tag-referenced use, alongside what it could not read, and when GitHub's answer lacks sha_pinning_required still report the unpinned uses found, with tests for both cases"
---

abcd ahoy --remote can report the SHA-pinning advisory without its count of unpinned uses. The fidelity review of itd-2610290746077716 (rcp-6083de2cef52, ac-9 met with concerns) found that when any workflow or local action went unread, the switch takes the unread branch before the count branch, so the advisory names what was unread but drops the count of tag-referenced uses abcd did read (internal/core/ahoy/remote_baseline.go:913). An Actions answer that carries no sha_pinning_required field reads as unknown before the workflows are looked at (remote_baseline.go:911), so a repository with unpinned uses gets no advisory at all there.
