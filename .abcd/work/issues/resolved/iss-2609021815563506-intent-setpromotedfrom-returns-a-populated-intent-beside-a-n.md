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
resolution: "SetPromotedFrom and ErrBackEdgeTaken were replaced by AddRelatedIssue (48c61088c), which never returns a populated intent beside an error. Its return contract is now asserted at the primitive: the record's list on the append and the idempotent no-op paths, and the zero Intent beside every refusal."
impact: internal
resolved_by:
  commit: "c530b76645ebe2ed07bfe89c0cce02528e73c3f7"
---

intent.SetPromotedFrom returns a populated Intent beside a non-nil ErrBackEdgeTaken, deliberately and documented, and the promote route depends on that value to report the kept back-edge, but the primitive's own test discards the return, so the stated contract is asserted only indirectly through the capture result's BackEdgeKept field. The fidelity verdict for itd-2609020625400169 records this as its one missing item; a primitive-level assertion closes it.

## Grounds

- pursued: we expect primitive-level assertions on AddRelatedIssue's return to catch a regression the promote route's BackEdgeKept would otherwise hide; it is shown wrong if a mutation returning a populated intent beside an error, or an empty list on the no-op, passes the intent tests
