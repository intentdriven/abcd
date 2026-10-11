---
schema_version: 1
id: "iss-2609012111162089"
slug: "update-verb-never-proceeds-on-a-shadowed"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "ship-audit-itd-130-itd-132-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/update/update.go"
remedy: "Waits on BP: gap 1 needs the technical facilitator's ruling on whether `abcd update` proceeds on the owned entry an earlier PATH occupant shadows, reporting the shadow as spc-32 line 61 promised, or whether the shipped spec is amended to the delivered refusal (and, if amended, the refusal on the unprovenanced-regular-file shape also names the shadowed install, as the foreign shape already does). Gaps 2 and 3 are paid."
deferred_after: "v0.11.1"
deferral_reason: "a lane of its own, with one ruling inside it: gap 1 is spc-32's promise that update proceeds on a shadowed owned entry, which the delivered dispatch never reaches, so it is either built or the shipped spec is amended; gaps 2 and 3 are the missing non-TTY silence test, the .new cleanup after a failed copy, and the CA canary test, which that lane lands with it (drain lane drainRest, run A, 2026-09-29)."
---

Three gaps the itd-130 fidelity audit (receipt rcp-264f7b144576) found against spc-32. (1) spc-32 line 61 promised that on a shadowed entry the verb proceeds on the owned entry and reports the shadow; delivered dispatch targets only the first PATH occupant (ResolveUpdateTarget), so the 'update completes on a shadowed entry' path is unreachable, and when the first occupant is an unprovenanced regular file Plan drops LaterOwned so the refusal never mentions the shadowed working install. (2) The non-TTY silence criterion (ac-9) gates progress on stderr's TTY-ness rather than stdout's as written, and no test pins silence when piped (spc-32 line 93 promised one). (3) No test covers a failure after the download starts: mid-stream truncation is file-free only because minio/selfupdate buffers the body, and a copy failure into the .new file has no unlink path (spc-32 line 87 promised the cleanup test); the CA canary-read assertion at spc-32 line 78 is also absent (tests assert the env is unset instead).

## Progress 2026-09-30

Gaps 2 and 3 paid in 5375a21ac, 70467dda3 and 80c512a60 (lane drainUpdate, autonomous run A). Gap 1 waits on ruling BP, so the record stays open.

- Gap 2: the progress gate reads the stream progress is written to (stderr), and a piped-run test pins that stdout carries only the receipt (TestUpdatePipedPrintsNoProgress, with TestUpdateTerminalStderrGetsProgress as its counterpart). The criterion's stdout wording is recorded as loose rather than the code as wrong: iss-2609300015353414, resolved with an Audit Notes line on itd-130, criterion text untouched.
- Gap 3: Apply verifies the whole download before any file exists and stages the verified bytes itself, fsynced, keeping minio/selfupdate for the swap and rollback. Every failure after the staging file is opened unlinks it (TestApplyFailedStagingWriteLeavesNoNewFile, TestApplyFailedSwapLeavesNoNewFile; the swap case left `.abcd.new` behind before). The CA canary spc-32 line 78 promised is TestTransportConsultsItsPinnedRootPool and TestShippingConstructorNeverTrustsAPlantedCA.
