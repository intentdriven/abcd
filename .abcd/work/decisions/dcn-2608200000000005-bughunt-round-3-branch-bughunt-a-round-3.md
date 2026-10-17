---
id: dcn-2608200000000005
slug: bughunt-round-3-branch-bughunt-a-round-3
date: 2026-08-20
---
2026-08-20 — bughunt round 3 (branch bughunt-a/round-3): captured twelve findings (iss-360..iss-369, plus two renumbered to iss-373/iss-374 after round 2's own collision re-mint landed inside this round's block — the week's fourth collision,
each adversarially refuted before capture; ids minted above the sibling bughunt-b/round-2's iss-357 via
the cross-ref floor). Fixed eleven, one recorded-not-fixed. Code/security: iss-361 the lifeboat coverage
renderer (per-repo and aggregate) printed untrusted cross-repo status/tier/tiers_present/section fields
to the terminal with no termsafe — only the repo name was sanitised — so a crafted probe report injected
raw ANSI/OSC-8/bidi (reproduced); both renders now route every repo-derived string through sanitize with
widths from the sanitised strings; iss-366 termsafe.Sanitize masked the deprecated FEFF joiner but not
its Unicode successor U+2060 WORD JOINER, the U+2061-2064 invisible operators, or U+00AD SOFT HYPHEN —
all now masked; iss-364 the capture related_specs validator pinned the retired fn-N namespace (reFnID)
instead of the live spc-N, making the field unusable — switched to reSpcID, dropped the orphaned reFnID,
corrected issues/README.md. Surface (additive): iss-362 abcd banlist --json omitted the reach caveat its
command file tells an agent to relay verbatim — added a reach field to PrivateReport (unconditional,
mirroring the ahoy status board). Docs (fix): iss-363 commands/lint.md and prepare-this-repo.md
enumerated five lint conventions omitting the shipped identity-positioning rule; iss-367 commands/launch.md
documented a nonexistent top-level files count for launch --dry-run --json (the list is bundle.files).
Infra (fix): iss-365 the govulncheck CI job pinned @v1.1.4 whose bundled x/tools caps its type-checker at
go1.24 while go.mod declares go 1.25.6, so the scan failed at package loading and exited before analysing
anything — masked as an advisory red since it landed 2026-08-19; bumped to @v1.7.0 (verified locally to
load go1.25). Record (internal): iss-373 three ADRs (adr-11/13/20) ended with leaked tool-call scaffolding
(</content></invoke>) — deleted; iss-374 the ADR index listed adr-44 proposed vs the record's accepted —
corrected; iss-360 adr-7 carried no frontmatter against the ADR contract (sole outlier of 36) — added;
iss-368 the core/disembark glossary term linked its counterpart to the interview-context embark (a
different bounded context) rather than the /abcd:embark unpack surface — repointed. Ledger hygiene:
resolved the stale iss-338 (round 2's persona-pronoun record already fixed by round 1's iss-321; all ten
cited sites are they/them). Recorded not fixed (stays open): iss-369 readTranscript opens the
hook-supplied transcript path without O_NOFOLLOW — the last bespoke external-input read not routed through
fsutil.ReadGuarded; refuted as no reachable exploit (harness-supplied path, no path policy, so a direct
path already reads any file), defence-in-depth only, left for a scoped consolidation. Model routing:
orchestration on Claude Fable 5; five parallel hunters, six per-finding refuters, and one of the two
pre-merge reviewers on Claude Opus 5; the second pre-merge reviewer on Claude Fable 5 (dual-model merge gate).
