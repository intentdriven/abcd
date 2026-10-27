---
schema_version: 1
id: "iss-2610262031459784"
slug: "the-press-release-says-abcd-ahoy-install"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 173 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/01-press-release.md:25"
remedy: "none (filed automatically)"
related_intents: [itd-83]
finding_ref: cfk-19959992e52d2edc
---

The press release says `/abcd:ahoy install` writes the AGENTS.md marker block, but itd-83 says `prepare-this-repo` owns that section.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/01-press-release.md:25` — "`/abcd:ahoy install` bootstraps abcd in any repo (transparent prompts, visibility-driven gitignore, marker block in AGENTS.md, prompt-router hook)"
- End B: `.abcd/development/intents/drafts/itd-83-review-bar-fires-itself.md:67` — "of `AGENTS.md` that `prepare-this-repo` already owns"

01-press-release has `/abcd:ahoy install` bootstrap a repository, including its marker block in AGENTS.md. itd-83 puts its fallback pointer in the marked conventions section of AGENTS.md 'that `prepare-this-repo` already owns'. The brief registers no `prepare-this-repo` surface, so the one setup surface that owns the marker block has two names.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 173.
