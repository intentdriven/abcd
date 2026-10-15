---
id: dcn-2608190000000003
slug: bughunt-round-1-branch-bughunt-b-round-1
date: 2026-08-19
---
2026-08-19 — bughunt round 1 (branch bughunt-b/round-1): captured iss-291..iss-304 and fixed
ten. Code: iss-291 the guard exec-string cluster reader ignored value-taking short flags, so
`script -Tc out.txt -c '<blocker>'` was a silent allow that also disarmed the Tier-2 fail-safe
(value flags now abort the cluster read); iss-293 ParseSemver swallowed strconv.Atoi's range
error, clamping an oversized component to MaxInt64 (now propagated, matching spec.go); iss-292
the wrapper-probe test wrote junk files (0,1,PATH,root) into the tracked tree because runsCommand
had no cmd.Dir (now a temp dir; the four committed artefacts removed). Docs/record: iss-294
version --check "only network" claim, iss-295 README unshipped brief-onboarding claim, iss-296
ADR-index + ruleset-README count drift, iss-297 adr-40 verb-rename residue in roadmap+brief,
iss-298 all 18 broken/stale brief heading anchors and section citations (anchorcheck now 0),
iss-299 phantom "in-session dispatch" internals chapter, iss-300 persona glossary role-first
contradiction. Recorded not fixed (stay open): iss-301/iss-302 external-review.yml approval-tally
(paginated per-page --jq) and error-masking (|| echo none + under-permissioned collaborators
endpoint) — required-check workflow, unverifiable without a real Actions run; iss-303 links_resolve
never validates heading anchors (systemic fix behind the anchor cluster); iss-304 deferred nitpicks
(a-4 memory ingest --source size TOCTOU, b-4 completion/help omitted from the CLI reference, d-12
the CI job id `record-lint` runs the reviews-charter script). Refuted this round: the scripts/
payload over-inclusion (knowingly-deferred open dependency, iss-34), the attribution/reviews gates
running from the PR tree (CODEOWNERS contains it), AGENTS.md's CI-job list (ci.yml-scoped, iss-182),
the release Go-version float (deliberate), and the 05-personas.md random-picker line (prior art iss-49).
