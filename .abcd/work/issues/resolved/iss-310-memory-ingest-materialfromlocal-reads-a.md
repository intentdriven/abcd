---
schema_version: 1
id: "iss-310"
slug: "memory-ingest-materialfromlocal-reads-a"
severity: "nitpick"
category: "security"
source: "agent-finding"
found_during: "bughunt-round-1"
found_at: "internal/core/memory/ingest.go"
resolution: "memory ingest reads via fsutil.ReadGuarded, closing the swap window and uncapped-growth read."
impact: fix
---

memory ingest materialFromLocal reads a user-typed operand with Stat-then-ReadFile leaving a swap window and an uncapped-growth read
## Evidence
`internal/core/memory/ingest.go:601-618` — `os.Stat` → IsRegular/size checks → `os.ReadFile(resolved)`. The size cap is checked only against the Stat result; `os.ReadFile` is itself uncapped, so a regular file that GROWS after the Stat is read whole. A type/symlink swap in the window is not refused on the same fd. burst-6 claimed "EVERY CLI operand read now uses the one-call guarded primitive" — overstated by exactly this core-side site (`abcd memory ingest <path>` is a plugin-surface operand, `commands/memory.md:31`).

## Adversarial verdict: CONFIRMED (nitpick)
Static-plant FIFO/device already refused by the IsRegular pre-check (not the same class as iss-202). Residual is the TOCTOU swap window + uncapped-growth read. `EvalSymlinks` at :591 fully resolves chains, so `fsutil.ReadGuarded(resolved, maxFetchBytes)` (O_NOFOLLOW on the leaf) works without breaking the legitimate symlink-source case. Fix: swap the read for ReadGuarded, keep the Stat pre-check (belt-and-suspenders, per iss-109), map sentinels onto existing messages. Repo precedent (readLessonsPayload/readSynthesisPayload in burst-6) is to convert.
