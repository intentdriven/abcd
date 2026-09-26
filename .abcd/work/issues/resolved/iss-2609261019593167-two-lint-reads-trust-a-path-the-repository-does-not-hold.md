---
schema_version: 1
id: "iss-2609261019593167"
slug: "two-lint-reads-trust-a-path-the-repository-does-not-hold"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-lintA item 1 sibling sweep"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/prosecitations.go"
resolution: "loadProseBaseline reads the configured baseline through readRepoLeaf (lexical containment, its directory resolved inside the repository, and fsutil.ReadGuardedInRoot so the leaf is never followed), so a baseline outside the tree is a configuration refusal rather than an exemption list; ReadReadingOutstanding reports a store root that resolves outside the repository as one unsafe entry and reads nothing below it. TestProseCitationBaselineIsReadOnlyInsideTheRepository and TestReadingWalkRefusesASymlinkedStoreRoot were watched failing before the change."
impact: fix
resolved_by:
  commit: "eeab4d69"
---

Two lint reads trust a path the repository does not hold, siblings of the receipt gate's symlinked commit directory (iss-2609261016494611), found on its sweep. (1) loadProseBaseline (internal/core/lint/prosecitations.go) reads the prose-citation baseline at the committed config's baseline path with no containment at all: a path that climbs out with '..', or one whose directory is a link out of the tree, is read, and every id the out-of-tree file names stops firing, so prose_citation_resolves passes on an exemption list the tree does not hold. (2) ReadReadingOutstanding (internal/core/lint/readingoutstanding.go) refuses a link at every directory below the issue store and not at the store root itself, so a symlinked store root carries the whole reading walk out of the tree and the outstanding board reports on records the repository does not hold.

## Grounds

- pursued: no lint read that a configured or store path names is satisfied by content outside the repository; an out-of-tree baseline still exempting an id, or a linked store root the reading walk still reads, would show it wrong
