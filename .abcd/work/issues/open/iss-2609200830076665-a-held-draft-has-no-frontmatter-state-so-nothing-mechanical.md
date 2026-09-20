---
schema_version: 1
id: "iss-2609200830076665"
slug: "a-held-draft-has-no-frontmatter-state-so-nothing-mechanical"
severity: "minor"
category: "future-work-seed"
source: "agent-finding"
found_during: "Gropius sub-agent-lane experiment, session gropiusllm-2b, relayed to abcd-17 on 2026-09-20"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/lifecycle.go"
---

A held draft has no frontmatter state, so nothing mechanical stops a lane from planning it. A hold on an intent draft is prose (a Review or Open Questions section saying it is held and why); the intent frontmatter the loader reads carries no hold field, and abcd intent plan checks the draft's sections and the bucket, never a hold. The surface page makes plan a human sign-off act that is never run unattended, which is the convention that holds today, and a sub-agent lane that follows its own brief rather than the page has nothing in its way. Relayed from the Gropius session gropiusllm-2b on 2026-09-20, which ran five record-only lanes on held drafts with a written prohibition on planning as the only guard. Wanted: a held: <reason> frontmatter key on a draft, written by a verb and refused as free text, with plan refusing a held draft and naming the reason, and the record dispatcher (abcd <itd-N>) reporting the hold as the next move. Sibling of iss-2609170726457256 (no verb stamps judged fields on a draft): this is the state field, that one the judgement fields.
