---
schema_version: 1
id: "iss-2610262035538469"
slug: "the-site-internals-page-says-abcd-site"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 605 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/10-site.md:34"
remedy: "none (filed automatically)"
finding_ref: cfk-aac2aa77d6ad8652
---

The site internals page says `abcd site build` writes into `site/` and nowhere else; the site chapter names a write outside the output directory and an output directory the caller chooses

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/10-site.md:34` — "copies rasters verbatim, and writes into `site/` and nowhere else"
- End B: `.abcd/development/brief/04-surfaces/22-site.md:195` — "The one write outside the output directory is the missing-label completion of the interface-string file above"

10-site.md says the build 'writes into `site/` and nowhere else'. 22-site.md says the build's one write outside the output directory is the missing-label completion of the interface-string file (backed by its own ADR), and that both write paths can be pointed at a different directory than `./site`. A builder holding to the internals page would leave out the interface-string write and fix the output path.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 605.
