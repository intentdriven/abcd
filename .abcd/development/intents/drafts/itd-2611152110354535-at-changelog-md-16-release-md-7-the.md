---
id: itd-2611152110354535
slug: at-changelog-md-16-release-md-7-the
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2610151422150430]
origin: extracted-from-record
production_mode: hand-written
---

# At CHANGELOG.md:16; RELEASE.md:7: The entry says that with `oracle.review = rp` in the repository's or the machine's configuration, a build lane sends its reviews to RepoPrompt. In the code (internal/core/oracle/review.go) the route also needs `oracle.rp.url`, and only ~/.abcd.noindex/config.json may set it: a repository that sets it is refused. With only `oracle.review = rp`, every review falls back to the host. commands/build.md and commands/implement.md state this correctly; the changelog line does not mention the URL. (v0.14.2 release docs-currency review, raised by opus.)

## Press Release

> _Seeded by promotion from iss-2610151422150430. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2610151422150430`: At CHANGELOG.md:16; RELEASE.md:7: The entry says that with `oracle.review = rp` in the repository's or the machine's configuration, a build lane sends its reviews to RepoPrompt. In the code (internal/core/oracle/review.go) the route also needs `oracle.rp.url`, and only ~/.abcd.noindex/config.json may set it: a repository that sets it is refused. With only `oracle.review = rp`, every review falls back to the host. commands/build.md and commands/implement.md state this correctly; the changelog line does not mention the URL. (v0.14.2 release docs-currency review, raised by opus.). Read that issue record for the source observation.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
