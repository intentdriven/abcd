---
schema_version: 1
id: "iss-2609211735419853"
slug: "a-record-slug-is-cut-mid-token-at-the-cap-so-ac-10-reads-as-ac-1"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "Dessau pilot 3, relayed by session gropiusllm-34 on 2026-09-21"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/roots.go deriveSlug; internal/core/intent/create.go deriveIntentSlug; internal/core/decide/decide.go deriveSlug"
---

A record slug is cut mid-token at the sixty-character cap, so a slug that ends in a numbered token reads as a different token: a capture whose text named acceptance criterion ac-10 landed as ...-ac-1, and one naming ac-13 the same, which reads as a capture about ac-1 and makes two distinct records look like one in a directory listing. The cut is collapsed[:60] followed by a hyphen trim, in three places that each derive a slug the same way and cap it the same way: the capture ledger, the intent store and the decision store. Wanted: one slug cap in the record-id package, the one home for record filenames, that cuts at the last token boundary at or before the cap (and only falls back to a hard cut when the first token alone exceeds it), with the three sites calling it; a slug never ends in a token the text did not contain.
