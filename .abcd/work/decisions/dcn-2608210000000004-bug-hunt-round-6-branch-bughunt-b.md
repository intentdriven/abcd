---
id: dcn-2608210000000004
slug: bug-hunt-round-6-branch-bughunt-b
date: 2026-08-21
---
2026-08-21 — Bug-hunt round 6, branch bughunt-b/round-6. Baseline (make preflight
+ gofmt -l .) green before any change and on the final tree. Five parallel Opus 5
hunters swept the four dimensions; each candidate was adversarially refuted by an
independent Opus 5 subagent before capture. Findings: 8 substantive + 2 confirmed
doc/CLI, 5 refuted, 1 (I2) reverted on implementation. Fixed and resolved this
round (each behaviour-changing code/script fix carries a test watched fail
before the change, the lifeboat probe size-TOCTOU excepted — that race is closed
by construction and its boundary test guards the cap, not the race):
iss-2608211849467013 the
guard tokenizer did not know shell redirection operators, so a glued redirection
(git push --force>/dev/null) mutated the flag token and the blocker missed — a
silent allow — and a leading redirection degraded a Tier-1 block to a warn; the
tokenizer now drops a redirection and its target. iss-2608211849463840 the
record/docs lint reporting walk read cfg.Roots with a raw os.ReadFile and no
containment while the sibling citation collector guards the same field, so a
cloned repo's committed lint config could read/lint/follow a file outside the
tree (reproduced: escaping root, symlinked leaf, /dev/zero OOM) — routed through
containedRepoPath/resolvedInsideRoot/containedRealPath/ReadGuarded, and the
sibling glossary_dir walk in loadForbiddenSynonyms (the same cloned-repo-config
read path, caught by both pre-merge reviews) was swept through the same stack;
the remaining config-derived record reads in the lint package are tracked as a
follow-up (iss-2608211914592726).
iss-2608211849461061 frontmatter.Fields requires --- on line 0, so a comment-led
record (13 glossary term files) slipped the no_git_metadata blocker; the
lint-local frontmatterFields now slices past the comment, preserving absolute
line numbers. iss-2608211849466878 citation DaysBetween took each date's local
calendar day and restamped UTC, making staleness and the release-gate overdue
blocker timezone-dependent; both ends now convert to UTC first. iss-2608211849468791
the attribution gate's trailer/None presence checks are line-end-anchored over a
class excluding CR, so a CRLF web-UI pull-request body false-red a correct
trailer on the required check; check_text now normalises CRLF before every rule,
with CRLF corpus cases. iss-2608211850070541 identity init faults exit 2 (was 1);
iss-2608211850074600 the lifeboat probe read cap+1 and refuses a file grown past
the cap (size TOCTOU); iss-2608211850070318 history list --json emits [] not null.
Doc/record: iss-2608211849581263 + iss-2608211849583757 the persona brief said
the picker chooses at random and omitted Nia (a residual of resolved iss-300 —
selection is by role, never by name; roster is fourteen); iss-2608211849583208
the bundled INTENTS default persona rule reached this repo's own intent authors
because .abcd/rules.json never overrode INTENTS, so an INTENTS override now points
authors at the registry (the default stays correct for a registry-less adopter);
iss-2608211849580624 the README described download-on-every-update and a no-op
.binary-meta remedy, rewritten for the spc-35 persistent cache; iss-2608211849582190
CONTRIBUTING granted the fenced-quotation carve-out for commit messages too, now
scoped to the pull-request body. Refuted / considered-and-rejected (kept out of
the ledger): R3 the release-gate manifest prompt naming ten of twenty commands
(an illustrative grounding string, not a gated inventory, and editing it ripples
into receipt hashes); R4/R5 related_adrs and prose citations of superseded ADRs
(resolve via the retired-set mechanism, deliberate historical provenance); R6
"11 adapters" vs "five capability seams" (implementations vs categories, not a
contradiction); R7 the adr-13 bare path to itd-36's old bucket (a benign ungated
breadcrumb; the authoritative related_intents resolves by id); the update.go
redirect-host case-fold (fails closed, availability only, impractical to test
through a loopback redirect). I2 (release.yml "dormant until the public flip"
comments) was implemented then reverted: release.yml is a scaffold template
rendered for adopters who may be private, so the visibility-flip framing is
generic-correct there, unlike the abcd-specific runbook round 4 corrected. Model
routing: orchestration on Claude Fable 5; five parallel hunters and eight
per-finding adversarial refuters on Claude Opus 5; the dual pre-merge review runs
one reviewer as Claude Opus 5 and one as Claude Fable 5.
