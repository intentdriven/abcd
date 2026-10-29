---
schema_version: 1
id: "iss-2610262033526755"
slug: "the-board-s-six-section-design-reads-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 444 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/08-abcd.md:543"
remedy: "none (filed automatically)"
finding_ref: cfk-53c3325912099409
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The block quote opening that part of 08-abcd.md says the six-section board does not exist on any shipped surface and names the logbook as unread; the retired-location detector in the build sequence is the later fact."
---

The board's six-section design reads a recent-logbook section; the build sequence says the logbook location is retired and a detector fails the build on any internal Go source that names it.

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/08-abcd.md:543` — "**Recent logbook**: the last five entries by modification time"
- End B: `.abcd/development/brief/06-delivery/01-build-sequence.md:176` — "`.abcd/logbook/` is a retired location, held retired by an armed detector: `TestNoRetiredLogbookLocationInSource` fails the build if any non-test Go source under `internal/` so much as names it."

08-abcd's design target renders section 4, 'Recent logbook: the last five entries by modification time', from the core renderer internal/core/board, and lists only the last-disembark and dev-sync sections as open questions. 01-build-sequence says .abcd/logbook/ is retired and TestNoRetiredLogbookLocationInSource fails the build if any non-test Go source under internal/ names it. Section 4 cannot be built as designed without tripping that detector, and the design does not record that it has lost its source.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 444.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The block quote opening that part of 08-abcd.md says the six-section board does not exist on any shipped surface and names the logbook as unread; the retired-location detector in the build sequence is the later fact.
