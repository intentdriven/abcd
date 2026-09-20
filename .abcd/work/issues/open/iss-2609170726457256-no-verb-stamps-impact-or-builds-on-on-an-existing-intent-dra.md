---
schema_version: 1
id: "iss-2609170726457256"
slug: "no-verb-stamps-impact-or-builds-on-on-an-existing-intent-dra"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "Gropius managed-repo session gropiusllm-56, relayed to abcd-17 on 2026-09-17"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/cli.go"
---

No verb stamps impact or builds_on on an existing intent draft. --impact exists only on the filing call (abcd intent "<text>" --impact) and again at the close that ships the intent (spec close --impact); between those two points a draft that was filed without a judgement has no verb that adds one, and builds_on, a schema-known frontmatter field the readers walk, has no flag on any verb at all. Relayed from the Gropius managed-repo session gropiusllm-56 on 2026-09-17, where two sessions hand-edited the frontmatter of five drafts with sed to set both fields, which bypasses every validator the verbs carry (ParseImpact, the never-internal rule for intents, the record-id shape of a builds_on target). Confirmed against the v0.9.0 command tree: intent plan takes only --production-mode, intent link writes spec_id only, and no verb names builds_on. Sibling records: iss-126 (resolved) gave impact a home at the close, and iss-2609091256264547 (open) covers the three typed relations the schema lacks; this finding is about the one relation it has and the one judgement it accepts, neither of which a verb can write onto a draft. Intent-shaped by the itd-84 four-piece table (a capability: a draft's judged fields are settable through a verb after filing), so the routing to an intent draft is the human's to confirm; recorded here first.

**Corroboration (2026-09-20, Gropius session gropiusllm-97, relayed to
abcd-17).** Third session, same hand edit, at the exact point the judgement
is made: the planning interview settles the impact class, and `abcd intent
plan` takes no `--impact`, so the value was stamped into the draft's
frontmatter by hand before planning. Create and close carry the flag; plan,
the verb that runs when the judgement is actually made, does not.
