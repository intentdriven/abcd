---
id: itd-152
slug: autonomous-cloud-runs-in-sibling-repos
spec_id: spc-45
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-178]
impact: additive
---

# Autonomous cloud runs in sibling repos leaked harness attribution footers and a live session URL into public GitHub artifacts: the harness auto-appends a 'Generated with' footer plus a session link when a PR or issue is created, overriding the repos' Assisted-by-only attribution policy (commit messages stayed clean; the leak surface was PR bodies and issue comments, plus GitHub's public edit history retaining the pre-scrub revision). Leak shape only - no session ids reproduced here. Two remedies needed: (a) every autonomous routine prompt must ban session URLs and harness footers in public text AND mandate a post-create re-read-and-strip of every PR/issue/comment the loop creates, because the append happens outside the model's own text; (b) abcd should detect the class - session-URL and harness-footer patterns belong with the shared privacy pattern set (iss-154 family / itd-74 banlist territory) so audit and docs-lint flag them in any committed or posted text.

## Press Release

> _Seeded by promotion from iss-178. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-178`: Autonomous cloud runs in sibling repos leaked harness attribution footers and a live session URL into public GitHub artifacts: the harness auto-appends a 'Generated with' footer plus a session link when a PR or issue is created, overriding the repos' Assisted-by-only attribution policy (commit messages stayed clean; the leak surface was PR bodies and issue comments, plus GitHub's public edit history retaining the pre-scrub revision). Leak shape only - no session ids reproduced here. Two remedies needed: (a) every autonomous routine prompt must ban session URLs and harness footers in public text AND mandate a post-create re-read-and-strip of every PR/issue/comment the loop creates, because the append happens outside the model's own text; (b) abcd should detect the class - session-URL and harness-footer patterns belong with the shared privacy pattern set (iss-154 family / itd-74 banlist territory) so audit and docs-lint flag them in any committed or posted text.. Read that issue record for the source observation.

## Scope Conditions

None stated.

## Acceptance Criteria

- **Given** an outbound PR body that carries a harness-appended attribution footer, **when** the scanner evaluates the artefact before it is posted, **then** the footer is caught and stripped so the posted text carries the repo's `Assisted-by:`-only attribution.
- **Given** an outbound issue comment containing a live session URL, **when** the scanner evaluates it, **then** the session URL is caught, whether the model authored it or the harness appended it.
- **Given** an outbound artefact whose text is already clean, **when** the scanner evaluates it, **then** it passes unchanged with no finding.
- **Given** the shared privacy pattern set, **when** it is applied by `abcd audit` and `docs-lint`, **then** the session-URL and harness-footer patterns are flagged in any committed or posted text, not only in freshly created PR bodies.
- **Given** an autonomous routine prompt assembled for a managed repo, **when** the prompt is composed, **then** it carries the policy that bans session URLs and harness footers in public text and mandates a post-create re-read-and-strip of every PR, issue and comment the loop creates.

## Open Questions

_None recorded yet._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-121c23aff688 -->
Fidelity review — receipt rcp-121c23aff688 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:ed2f5826abc3017891160f73694ff0065239b83c9b5243afb4f8a12fb26f0cb6
Input attestations: diff:bd4b49948 6cc3d571a (spc-45) + 8c1e2ff42 f93acc247 (spc-2609230612443069), judged in the tree at de42f275a@sha256:f4acf2490853d5a3f188a1eb7226ae5b22642d54897563142ac740a809fc0f3a;

Acceptance rollup: MET 1 · MET_WITH_CONCERNS 4 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: ScrubOutbound drops the footer's whole line and keeps the Assisted-by trailer, pinned by TestScrubOutboundStripsFooterKeepsTrailer; the concern is that nothing in the tree calls it before a post (spc-45 scopes the forge client out, iss-2608282124231752 open and deferred), so the 'before it is posted' half rests on the routine obeying OutboundPolicy, not on the scanner running
  evidence: internal/adapter/scanner/outbound.go:53 — "func ScrubOutbound(repoRoot, text, label string) (string, []Finding, error)"
  evidence: internal/adapter/scanner/outbound.go:83 — "if f.Kind == kindHarnessFooter { drop[f.Line] = true }"
  evidence: internal/adapter/scanner/harnessleak_test.go:185 — "TestScrubOutboundStripsFooterKeepsTrailer ... if strings.Contains(got, "Generated with") ... if !strings.Contains(got, "Assisted-by: Claude:some-model")"
  evidence: .abcd/work/issues/open/iss-2608282124231752-scanner-scruboutbound-has-no-front-door.md:16 — "no CLI command, plugin verb or routine prompt calls it"
- ac-2 — MET_WITH_CONCERNS: harness_session_url matches a session URL by shape, origin-blind by design, and TestScrubOutboundCatchesSessionURL shows a comment's URL caught and masked; the concern is the stated coverage boundary: an all-lower-case non-hex id is not detected (hasOpaqueSessionID), so a harness minting that shape is outside the criterion's 'is caught'
  evidence: internal/adapter/scanner/harnessleak.go:58 — "harnessSessionURLRe = regexp.MustCompile(`(?i)\bhttps?://...session[_/=-] (?:id=)?[A-Za-z0-9-]{12,}`)"
  evidence: internal/adapter/scanner/harnessleak.go:14 — "A detector cannot tell the two apart after the fact, and does not need to: the shape is the finding, whoever wrote it."
  evidence: internal/adapter/scanner/harnessleak_test.go:205 — "TestScrubOutboundCatchesSessionURL ... if strings.Contains(got, id) { t.Errorf("session URL survived the scrub""
  evidence: internal/adapter/scanner/harnessleak.go:219 — "COVERAGE BOUNDARY ... An id that is all lower-case and not all hex ... is NOT detected"
- ac-3 — MET: ScrubOutbound returns the input verbatim with nil findings when the scan is empty, and TestScrubOutboundPassesCleanTextUnchanged asserts byte equality and zero findings; CheckOutbound's twin test passes clean text too
  evidence: internal/adapter/scanner/outbound.go:69 — "findings := sc.ScanText(text, label); if len(findings) == 0 { return text, nil, nil }"
  evidence: internal/adapter/scanner/harnessleak_test.go:223 — "TestScrubOutboundPassesCleanTextUnchanged ... if got != body { t.Errorf("clean text was rewritten""
  evidence: internal/adapter/scanner/outbound_check_test.go:53 — "func TestCheckOutboundPassesCleanText"
- ac-4 — MET_WITH_CONCERNS: the class is one definition (HarnessLeakPatterns) selected by the repo-conformance privacy rule over tracked files and by the record/docs lint's harness_leak rule, each with a positive test; two concerns: the verb the criterion names, `abcd audit`, does not exist at BASE (the surface is `abcd lint`, with `abcd lint outbound` for one posted artefact), and the docs-lint rule is inert unless a config arms it (TestHarnessLeakDisabledByDefault) though the ahoy seed and this repo's docs-lint.json/record-lint.json arm it as a blocker
  evidence: internal/core/repolint/rule_privacy.go:148 — "leakPatterns := scanner.HarnessLeakPatterns()"
  evidence: internal/core/repolint/rule_privacy_harness_test.go:31 — "func TestAC_PrivacyHarnessLeakInCommittedFile"
  evidence: internal/core/lint/harnessleak.go:86 — "for _, p := range scanner.HarnessLeakPatterns()"
  evidence: internal/core/lint/harnessleak_test.go:40 — "TestHarnessLeakInLintedProse ... expected both leak shapes to be flagged"
  evidence: internal/core/lint/harnessleak_test.go:99 — "TestHarnessLeakDisabledByDefault ... expected an unconfigured rule to be inert"
  evidence: .abcd/docs-lint.json:303 — ""harness_leak": { "enabled": true, "severity": "blocker" }"
  evidence: internal/surface/cli/lint.go:25 — "Use: "lint""
  evidence: internal/surface/cli/lint_outbound.go:72 — "Use: "outbound [FILE]""
- ac-5 — MET_WITH_CONCERNS: renderBrief quotes scanner.OutboundPolicy verbatim under its own heading before the record, and TestTheBriefCarriesTheOutboundPolicy pins it once and ahead of the first record block; the concern is scope: the only prompt abcd composes is `abcd build`'s lane brief, so an autonomous routine assembled outside that loop (the cloud routines of iss-178, the itd-107 template still in drafts/) receives nothing from abcd
  evidence: internal/core/implement/loop/brief.go:518 — "p("## Outward-facing text\n\n") ... p("> %s\n\n", scanner.OutboundPolicy)"
  evidence: internal/core/implement/loop/brief_test.go:177 — "if n := strings.Count(brief, scanner.OutboundPolicy); n != 1"
  evidence: internal/adapter/scanner/outbound.go:28 — "const OutboundPolicy = "Never put a live session URL or a tool's own attribution footer ... After creating ANY pull request, issue or comment, re-read what was actually created and strip"
  evidence: .abcd/development/intents/drafts/itd-107-autonomous-routines-assemble-from-one.md:172 — "directs a post-create re-read-and-strip of each pull request, issue, and"

Gap audit:
- honoured:
  - session-URL and harness-footer patterns belong with the shared privacy pattern set, defined once
    evidence: internal/adapter/scanner/patterns.go:248 — "return append(p, HarnessLeakPatterns()...)"
    evidence: internal/adapter/scanner/harnessleak_test.go:164 — "func TestHarnessLeakPatternsAreInTheCanonicalSet"
  - audit and docs-lint flag the class in committed text
    evidence: internal/core/repolint/rule_privacy_harness_test.go:31 — "func TestAC_PrivacyHarnessLeakInCommittedFile"
    evidence: internal/core/lint/harnessleak_test.go:172 — "func TestRecordLintArmsHarnessLeakOverTheLedger"
  - an outbound artefact is scrubbed and a clean one passes unchanged
    evidence: internal/adapter/scanner/harnessleak_test.go:185 — "func TestScrubOutboundStripsFooterKeepsTrailer"
    evidence: internal/adapter/scanner/harnessleak_test.go:223 — "func TestScrubOutboundPassesCleanTextUnchanged"
  - the prompt an autonomous build hands its implementer bans the shapes and mandates the post-create re-read-and-strip
    evidence: internal/core/implement/loop/brief.go:521 — "p("> %s\n\n", scanner.OutboundPolicy)"
    evidence: commands/build.md:108 — "the outbound policy: no session URL or tool attribution footer in public text, and a re-read-and-strip"
- diverged:
  - 'every autonomous routine prompt' carries the policy — delivered for abcd build's lane brief only; a routine assembled outside that loop (the cloud runs of iss-178) gets nothing, and the routine template is itd-107, still a draft
    evidence: internal/core/implement/loop/brief.go:511 — "The outbound policy (itd-152): ... the prompt that starts the agent is where the rule has to be."
    evidence: .abcd/development/specs/closed/spc-2609230612443069-routine-prompts-carry-the-outbound.md:24 — "no routine-prompt composer embeds it; only lint findings and reports quote it"
    evidence: .abcd/development/intents/drafts/itd-107-autonomous-routines-assemble-from-one.md:172 — "directs a post-create re-read-and-strip"
  - 'the scanner evaluates the artefact before it is posted' — the scrub is a primitive with no caller; the posting-time control is the policy text the routine obeys (recorded as iss-2608282124231752, open, deferred through v0.11.1)
    evidence: .abcd/work/issues/open/iss-2608282124231752-scanner-scruboutbound-has-no-front-door.md:16 — "no CLI command, plugin verb or routine prompt calls it, and internal/ cannot be imported from outside the module"
    evidence: internal/adapter/scanner/outbound.go:6 — "abcd opens no pull requests and owns no forge client, and this does not change that (spc-45)"
  - 'abcd audit' — the verb at BASE is 'abcd lint' (privacy-hygiene rule) plus 'abcd lint outbound' for one artefact; no top-level audit verb exists
    evidence: internal/surface/cli/lint.go:25 — "Use: "lint""
    evidence: internal/surface/cli/lint_outbound.go:3 — "The front door onto scanner.CheckOutbound: `abcd lint outbound`."
- missing: (none)
<!-- abcd-review-end receipt=rcp-121c23aff688 -->
