---
schema_version: 1
id: "iss-2610262038472526"
slug: "itd-14-marks-a-prompt-s-version-with-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 818 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-14-prompt-registry-versioning.md:29"
remedy: "Re-scope itd-14: drop the `## Prompt Version: X.Y` body field, since its own acceptance criteria already say `prompt_version: X.Y.Z` frontmatter and agent_contract already requires a semver `prompt_version` on every prompt; keep the changelog, diff-on-update and revert as the remainder, and close its open question on where the version lives."
related_intents: [itd-14]
finding_ref: cfk-424f102ee6423667
---

itd-14 marks a prompt's version with a `## Prompt Version: X.Y` field; 05-prompt-quality requires semver prompt_version frontmatter

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-14-prompt-registry-versioning.md:29` — "Per-prompt version field (`## Prompt Version: X.Y`)"
- End B: `.abcd/development/brief/05-internals/05-prompt-quality.md:41` — "`prompt_version` present and a valid semver, on every prompt."

itd-14 records a prompt's version in a `## Prompt Version: X.Y` body field. 05-prompt-quality requires a prompt_version frontmatter key holding a valid semver on every prompt, and agent_contract enforces it. The version marker has two names and two formats.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 818.
