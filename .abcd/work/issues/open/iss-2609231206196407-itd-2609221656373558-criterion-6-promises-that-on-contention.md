---
schema_version: 1
id: "iss-2609231206196407"
slug: "itd-2609221656373558-criterion-6-promises-that-on-contention"
severity: "minor"
category: "drift"
source: "user-observation"
found_during: "autonomous run 2026-09-23 fidelity audit"
origin: researcher-authored
production_mode: hand-written
---

itd-2609221656373558 criterion 6 promises that on contention of any kind the second session backs off and the log names the reason and the minutes spent. Delivered: a refused claim writes a backoff line with the reason but minutes fixed at 0 (internal/core/implement/claim.go:232), and a locked run state returns ErrContention from withLock with no log line at all (internal/core/implement/run.go:182-186), so lock contention is never counted in the derived comparison and the verb measures no backed-off minutes; queue contention is logged only when the session hand-writes implement log backoff. Found by the fidelity audit; not fixed here.
