---
schema_version: 1
id: "iss-2610262031466884"
slug: "context-names-abcd-ahoy-install-as-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 174 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/02-context.md:7"
remedy: "In itd-91 replace '/abcd:init / prepare-this-repo' with '/abcd:ahoy install'; neither old name is a registered surface and record-lint names ahoy install as the successor."
related_intents: [itd-91]
finding_ref: cfk-e58a2bfa9acc6de1
---

Context names `/abcd:ahoy install` as the install surface, but itd-91 puts its setup question in `/abcd:init` / `prepare-this-repo`.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/02-context.md:7` — "install / update abcd in any project (bootstraps configuration, gitignore, marker blocks, PATH symlink)"
- End B: `.abcd/development/intents/drafts/itd-91-ai-attribution-preference.md:31` — "in the onboarding surface (`/abcd:init` / `prepare-this-repo`)"

02-context says `/abcd:ahoy install` installs or updates abcd in any project and bootstraps its configuration. itd-91 puts its setup question in an onboarding surface it calls `/abcd:init` / `prepare-this-repo`. Neither name is a registered surface, and both refer to the setup that `ahoy install` already is.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 174.
