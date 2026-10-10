---
schema_version: 1
id: "iss-297"
slug: "adr-40-s-v0-6-0-verb-renames-intent"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "bughunt-round-1"
found_at: ".abcd/development/roadmap/phases/README.md"
resolution: "Swept intent review -> intent audit and disembark oracle -> review (and intent-fidelity-reviewer -> intent-auditor) across the roadmap phases and brief pages."
impact: fix
---

adr-40's v0.6.0 verb renames (intent review -> intent audit, disembark oracle -> disembark review) left stale residue in durable present-tense docs: roadmap/phases/{README.md,phase-0-substrate.md,phase-6-lifeboat.md} still say 'intent review', and brief/05-internals/08-skills.md plus brief/04-surfaces/README.md still list 'oracle' as a disembark sub-verb the binary now refuses
## Evidence

- `.abcd/development/roadmap/phases/README.md`, `.../phase-0-substrate.md` (two sites) and
  `.../phase-6-lifeboat.md` still write `/abcd:intent review` / `intent review ingest`, plus
  `phase-0-substrate.md`'s two `intent-fidelity-reviewer` mentions (renamed `intent-auditor`).
- `.abcd/development/brief/05-internals/08-skills.md` and
  `.abcd/development/brief/04-surfaces/README.md` still list `oracle` as a `disembark`
  sub-verb.
- The binary refuses both: `internal/surface/cli/cli.go` `retiredSubverbs`
  (`intent review -> audit`), pinned by `intent_surface_test.go`; `disembark oracle` refused,
  pinned by `disembark_synthesis_test.go`. Successors `intent audit` / `disembark review` ship.
- The roadmap and these brief pages are present-tense durable docs (`.abcd/development/README.md`
  scopes the chronological carve-out to `plans/` and `research/` only). Rename commit swept
  `brief/**` but missed these sites.

## Adversarial review

CONFIRMED (substantive) by an independent refuter: neither site is covered by iss-284's
release note or by the `surface_coverage` / brief-surface crosschecks (which don't reach
`roadmap/` or `05-internals/`). Fix: sweep the stale spellings to the shipped verb names.
