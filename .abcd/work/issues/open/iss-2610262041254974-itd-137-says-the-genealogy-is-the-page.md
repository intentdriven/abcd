---
schema_version: 1
id: "iss-2610262041254974"
slug: "itd-137-says-the-genealogy-is-the-page"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 999 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-137-maya-sees-how-every-decision-connects.md:34"
remedy: "none (filed automatically)"
related_intents: [itd-137]
finding_ref: cfk-396cb5dcd8cd865a
---

itd-137 says the genealogy is the page `/record/timeline/`; the site chapter says the genealogy is folded into the `/record/` dashboard, not served as a page of its own

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-137-maya-sees-how-every-decision-connects.md:34` — "`/record/timeline/` is the genealogy: five lanes"
- End B: `.abcd/development/brief/05-internals/10-site.md:55` — "the `/record/` dashboard rather than served as a page of its own"

itd-137's press release names `/record/timeline/` as the genealogy page with five lanes. The site chapter describes the genealogy as one static SVG folded shut into the `/record/` dashboard, not a page of its own. The genealogy cannot be both its own page and not one, so a reader following the shipped intent would look for, or link to, a route the site chapter says is not served.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 999.
