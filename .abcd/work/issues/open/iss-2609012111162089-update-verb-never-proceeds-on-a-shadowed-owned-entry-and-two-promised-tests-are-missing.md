---
schema_version: 1
id: "iss-2609012111162089"
slug: "update-verb-never-proceeds-on-a-shadowed-owned-entry-and-two-promised-tests-are-missing"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "ship-audit-itd-130-itd-132-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/update/update.go"
deferred_after: "v0.11.1"
deferral_reason: "a lane of its own, with one ruling inside it: gap 1 is spc-32's promise that update proceeds on a shadowed owned entry, which the delivered dispatch never reaches, so it is either built or the shipped spec is amended; gaps 2 and 3 are the missing non-TTY silence test, the .new cleanup after a failed copy, and the CA canary test, which that lane lands with it (drain lane drainRest, run A, 2026-09-29)."
---

Three gaps the itd-130 fidelity audit (receipt rcp-264f7b144576) found against spc-32. (1) spc-32 line 61 promised that on a shadowed entry the verb proceeds on the owned entry and reports the shadow; delivered dispatch targets only the first PATH occupant (ResolveUpdateTarget), so the 'update completes on a shadowed entry' path is unreachable, and when the first occupant is an unprovenanced regular file Plan drops LaterOwned so the refusal never mentions the shadowed working install. (2) The non-TTY silence criterion (ac-9) gates progress on stderr's TTY-ness rather than stdout's as written, and no test pins silence when piped (spc-32 line 93 promised one). (3) No test covers a failure after the download starts: mid-stream truncation is file-free only because minio/selfupdate buffers the body, and a copy failure into the .new file has no unlink path (spc-32 line 87 promised the cleanup test); the CA canary-read assertion at spc-32 line 78 is also absent (tests assert the env is unset instead).
