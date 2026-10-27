---
schema_version: 1
id: "iss-2610262040499773"
slug: "itd-14-versions-a-prompt-with-a-prompt"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 958 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-14-prompt-registry-versioning.md:29"
remedy: "none (filed automatically)"
related_intents: [itd-14]
finding_ref: cfk-ea8a807e4a54ee42
---

itd-14 versions a prompt with a '## Prompt Version: X.Y' heading; the verification matrix says each prompt carries a semver prompt_version field.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-14-prompt-registry-versioning.md:29` — "Per-prompt version field (`## Prompt Version: X.Y`)"
- End B: `.abcd/development/brief/06-delivery/02-verification-matrix.md:81` — "Every agent prompt carries a semver `prompt_version`, and the linter rejects a missing or malformed field"

itd-14 adds a per-prompt version as a body heading in X.Y form. The matrix's prompt-linter row says every agent prompt already carries a semver prompt_version, and the linter rejects a missing or malformed one. Those are two names and two formats for one value, and an X.Y value is malformed under the shipped rule.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 958.
