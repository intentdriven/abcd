---
schema_version: 1
id: "iss-2610262032225288"
slug: "ahoy-says-the-docs-target-defaults-to"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 258 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/03-embark.md:131"
remedy: "none (filed automatically)"
finding_ref: cfk-219e1bfdb5d40c98
---

Ahoy says the docs target defaults to skip, writing no conventions file; embark says no saved choice gives AGENTS.md

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/03-embark.md:131` — "`agents_md` or no choice saved gives `AGENTS.md`"
- End B: `.abcd/development/brief/04-surfaces/01-ahoy.md:1070` — "so the docs target defaults to `skip`: a default install writes it into none of the repository's committed conventions files"

Ahoy says the docs target defaults to skip, so a default install plants the marker block in none of the repository's conventions files and a project must name agents_md to get it. Embark reads docs.target with setup's reader yet maps no saved choice to AGENTS.md, so an embark into a target with no saved choice writes the block where a default install would write nothing. The two disagree on what the absent setting means.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 258.
