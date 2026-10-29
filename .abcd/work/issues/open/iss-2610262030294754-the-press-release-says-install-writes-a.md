---
schema_version: 1
id: "iss-2610262030294754"
slug: "the-press-release-says-install-writes-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 7 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/01-press-release.md:25"
remedy: "Reword the ahoy sentence in the press release (line 25) to what a default install does: scaffolds .abcd/ configuration and rules, applies the visibility-driven gitignore, writes the marker block only where --docs-target agents_md names it, and verifies (never writes) the hook manifest."
finding_ref: cfk-8dac513f145098ee
---

The press release says install writes a marker block in AGENTS.md and the prompt-router hook; the ahoy chapter says a default install writes the block nowhere

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/01-press-release.md:25` — "`/abcd:ahoy install` bootstraps abcd in any repo (transparent prompts, visibility-driven gitignore, marker block in AGENTS.md, prompt-router hook)."
- End B: `.abcd/development/brief/04-surfaces/01-ahoy.md:1070` — "so the docs target defaults to `skip`: a default install writes it into none of the repository's committed conventions files"

The press release says install bootstraps a marker block in AGENTS.md and the prompt-router hook. The ahoy chapter says the docs target defaults to `skip`, so a default install writes the block into no conventions file, and elsewhere says install only verifies the hook manifest and never writes it. The two disagree on what a default install leaves behind.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 7.
