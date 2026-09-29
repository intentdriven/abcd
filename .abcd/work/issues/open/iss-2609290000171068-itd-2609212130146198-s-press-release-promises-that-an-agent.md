---
schema_version: 1
id: "iss-2609290000171068"
slug: "itd-2609212130146198-s-press-release-promises-that-an-agent"
severity: "minor"
category: "drift"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: fidelity audit itd-2609212130146198"
found_at: "internal/surface/cli/guard_question.go"
origin: researcher-authored
production_mode: hand-written
---

itd-2609212130146198's press release promises that an agent cannot ask the human anything until it has said whom it is asking, but the delivered guard (criterion 2) covers only the host's question tool: a stop that asks in prose at the end of a turn is not gated, so the badge can still read abcd-managed while an answer is owed, which is the Mechanism's own falsifier. The intent's scope confines the guard to the question tool, so the press release overclaims; either the promise narrows to the tool or a stop-time gate closes the prose path. Found by the fidelity audit at 52c2236a5 (internal/surface/cli/guard_question.go questionTools).
