---
schema_version: 1
id: "iss-2609021815563506"
slug: "intent-setpromotedfrom-returns-a-populated-intent-beside-a-n"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "itd-2609020625400169 fidelity audit"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/lifecycle.go"
---

intent.SetPromotedFrom returns a populated Intent beside a non-nil ErrBackEdgeTaken, deliberately and documented, and the promote route depends on that value to report the kept back-edge, but the primitive's own test discards the return, so the stated contract is asserted only indirectly through the capture result's BackEdgeKept field. The fidelity verdict for itd-2609020625400169 records this as its one missing item; a primitive-level assertion closes it.
