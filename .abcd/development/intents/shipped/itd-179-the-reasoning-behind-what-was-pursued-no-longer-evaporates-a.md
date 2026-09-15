---
id: itd-179
slug: the-reasoning-behind-what-was-pursued-no-longer-evaporates-a
spec_id: spc-57
kind: standalone
suggested_kind: standalone
reclassification_history: []
builds_on: []
severity: minor
impact: additive
---

# The reasoning behind what was pursued no longer evaporates at the gate — readiness and triage record grounds for the conjecture, not only the decision

## Press Release

> **abcd records why a thing was pursued, at the moment it is pursued.**
> Grounds were recorded for deliberate non-action only — a wontfix carries a
> note, an ADR carries its alternatives — while the reasoning behind what
> went forward evaporated at the gate. Now the readiness gate and capture's
> triage routes take a grounds argument with a small vocabulary — `pursued`,
> `deferred`, `declined` — and the grounds name the conjecture being acted
> on, not merely the route taken. A triage without grounds is refused.

## What's In Scope

- A grounds argument on `intent ready` and on the capture triage routes
  (promote / resolve / wontfix), refusal on absence.
- The three-value vocabulary, with the grounds text free-form and naming the
  conjecture.

## What's Out of Scope

- Rewriting the ADR family's grounds — decision-granularity grounds stay
  where they are; this intent adds the finer grain beside them.

## Scope Conditions

None stated.

## Acceptance Criteria

- **Given** a capture routed to an intent draft, **when** triage runs
  without grounds, **then** the command refuses.
- **Given** a gate decision, **when** it is recorded, **then** the grounds
  name the conjecture, not only the decision.

## Grounds

- pursued: Pursued now because the reasoning behind what went forward is the half of the record that is never written down, and it is unrecoverable once the session ends: a wontfix keeps its note and an ADR keeps its alternatives, while every pursued conjecture leaves only its outcome. Recording it at the moment of pursuit is the only moment it is still known.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-ff0fec29d2cf -->
Fidelity review — receipt rcp-ff0fec29d2cf (verifier abcd:intent-auditor claude-opus-5[1m]).

Provenance: abcd:intent-auditor@claude-opus-5[1m] · rubric_hash sha256:542ed2cd51ff938717a3f47b2b332e8d47910beec0ca7ecdfd238ae7edf5ced5 · prompt_hash sha256:e5e158f517453eb879ac2931f683e6490b090b603c290856dde331bbcd2bbc0b
Input attestations: diff:932629f9...build/itd-179 (9bdf7d478f00dea3ce5f2d27430a09450588e06d)@sha256:2e28a1d9af6525ac95319cccdad5d6a6378179ee85a7a8a515af3287dad55b61;

Acceptance rollup: MET 0 · MET_WITH_CONCERNS 2 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: I did not read this one, I ran it. Against a binary built from the branch tip in a throwaway repo: `abcd capture promote iss-2608310903083914` with no --grounds exited 2, printed the usage refusal, and left `.abcd/development/intents/drafts/` EMPTY; the same call with `--grounds ""`, with an out-of-vocabulary token (`maybe:`), and with degenerate text (`pursued: pursued`, refused at the 3-word floor) each exited 2 and minted nothing; `capture resolve` without --grounds exited 2 and `capture wontfix` with an empty reason exited 2. The refusal is structurally ahead of the mint — requireGrounds runs before findIssue, before the lock, and before CreateDraft — which is why the residue is zero rather than merely usually zero. The happy path then wrote `- pursued: <text>` to the issue's `## Grounds`, and a later resolve APPENDED a second bullet rather than overwriting the first, so the promote-then-resolve sequence keeps both conjectures. CONCERN, two parts, and they are named rather than waved at. (i) SCOPE NARROWER THAN THE CRITERION'S 'a capture routed to an intent draft'. There are TWO routes that mint an intent draft, and only one is governed by this flag. `capture promote <rdi-N>` — a dispositioned reading item — dispatches at promote.go:109 BEFORE requireGrounds is ever reached, takes no --grounds, and I confirmed it mints a draft with the flag absent (TestCapturePromoteReadingItemNeedsNoGrounds, run green). It is not ungoverned — promoteReadingItem refuses without a disposition record, and the late find fixed before merge (9bdf7d47) makes it refuse an operand it would otherwise have accepted and discarded — but the record it delegates to carries `disposition_grounds` as FREE TEXT with no vocabulary token and no substance floor (cli.go:2819), which is a second, weaker grammar under the same flag name on the same verb. So on the rdi route the criterion holds by delegation to a laxer artefact, not by the mechanism this intent shipped. (ii) OVER-REFUSAL THAT MAKES THE ROUTE UNREACHABLE — iss-2608301908270888, open and unclosed. I reproduced it verbatim: `abcd capture "the loader drops rules when the < !-- abcd-review marker is stale..."` is accepted with no warning, and thereafter promote, resolve AND wontfix ALL refuse that record, permanently and identically, and `abcd capture --help` confirms there is no edit, amend or reopen verb to escape with. This does not falsify ac-1 — the criterion asks for refusal WITHOUT grounds, and this refuses WITH them too — so it is a concern, not a counterexample. But it means the triage route this criterion promises can be closed forever by ordinary captured prose. The delta fix made it survivable (the refusal names the construct and the body line instead of blaming the operator's grounds text; Promote mints no orphan — I checked the drafts directory after all three refusals and found only the one legitimate draft), and the body half was deliberately left live.
  evidence: internal/core/capture/promote.go:116 — "g, gRedacted, gDegraded, err := requireGrounds(repoRoot, \"promote\", req.Grounds)"
  evidence: internal/core/capture/promote.go:109 — "if reReadingItemID.MatchString(req.ID) {"
  evidence: internal/core/capture/promote.go:303 — "if strings.TrimSpace(req.Grounds) != \"\" {"
  evidence: internal/surface/cli/cli.go:2940 — "\"abcd capture \" + verb + \" requires --grounds \\\"\" + grounds.UsageSpelling() + \": <text>\\\" \""
  evidence: internal/surface/cli/cli.go:2819 — "dispositionCmd.Flags().StringVar(&dispGrounds, \"grounds\", \"\", \"disposition_grounds: why this answer (free text; required on every state except held)\")"
  evidence: internal/core/capture/grounds.go:80 — "return grounds.Grounds{Token: grounds.Declined, Text: folded}, 0, deg, nil"
  evidence: internal/core/grounds/record.go:223 — "return \"an HTML comment (`< !--` with no `-- >`)\""
  evidence: .abcd/work/issues/open/iss-2608301908270888-an-unclosed-comment-or-fence-in-an-issue-body-still-locks-th.md:37 — "There is no `edit`, `amend` or `reopen` verb. The only exit is a hand edit of a"
  evidence: internal/surface/cli/capture_surface_test.go:954 — "const run, item = \"rdg-2608300000000001\", \"rdi-2608300000000002\""
- ac-2 — MET_WITH_CONCERNS: The recording half is delivered and I exercised it end to end: `abcd intent ready itd-2` on a planned record with no grounds reported `[fail] grounds: no recorded grounds — the conjecture behind pursuing itd-2 is unrecorded` and exited 1; `--grounds "pursued: ..."` wrote the bullet into the record's `## Grounds` section and flipped the check to `[ ok ]` at exit 0; a second call APPENDED rather than replaced, so the earlier conjecture survives the later one; the closed vocabulary is enforced (`maybe:` refused, naming the three legal tokens); the substance floor holds on both halves and is genuinely script-aware — a 2-word/29-letter text was refused at the word floor while a Chinese-script text with no inter-word spaces at all was ACCEPTED, so scriptio continua is not punished for having one run; the migration is complete (zero `## Grounds (pursued)` stand-in sections remain on any spec, thirteen intents now carry the section); the record lint blocks a frontmatter `grounds:` key and says why; and the six user-facing spellings do render from `grounds.UsageSpelling()`/`ProseList()` rather than being typed. CONCERN, and it is the whole weight of this verdict: THE CRITERION'S ACTUAL OBSERVABLE IS NOT PRODUCED BY ANYTHING IN THE DELIVERED CODE. The criterion is not 'grounds are recorded' — it is that the grounds NAME THE CONJECTURE, NOT ONLY THE DECISION. I put the spec's OWN counterexample through the shipped gate: `abcd intent ready itd-2 --grounds "pursued: planned it because it is next"` — the exact phrase spc-57 uses to define a restatement of the decision — was accepted, written verbatim to the committed record, and reported by the gate as `[ ok ] grounds: 1 recorded ground(s), most recent pursued`, exit 0. Nothing anywhere refused it. The package concedes this in its own doc comment: no machine reads a sentence and knows which it is, so what ships is a FLOOR that 'claims nothing more'. The producer of the conjecture-naming property is therefore an INTERVIEW PROMPT, not a gate — `Ask for the expectation and its falsifier` at commands/intent.md:177 and commands/capture.md:139, both of which state outright that the floor 'cannot tell a conjecture' from a restatement. Applying itd-192: that producer DOES exist and IS wired in this phase, on both shipped surfaces, which is why this is MET_WITH_CONCERNS and not NOT_MET. But itd-192 also obliges me to name the intent that closes the gap, and I cannot: spc-57 puts semantic judgement out of scope and names NO successor, and itd-195 (filed as a stub on this branch) addresses executable claims about code behaviour, not the judging of grounds prose. So the conjecture-naming half rests on an unenforceable prose instruction with no scheduled closure. It demonstrably works when followed — itd-179's own entry names an expectation and why it is unrecoverable otherwise — but 'works when followed' is a discipline, not a gate, and this criterion was written as an outcome.
  evidence: internal/core/intent/ready.go:307 — "c.Detail = \"no recorded grounds — the conjecture behind pursuing \" + it.ID + \" is unrecorded\""
  evidence: internal/core/intent/ready.go:323 — "\"(vocabulary: \" + grounds.ProseList() + \") — name the conjecture being acted on, not the route taken\""
  evidence: internal/core/grounds/grounds.go:22 — "property and no machine reads a sentence and knows which it is, so what lives"
  evidence: internal/core/grounds/grounds.go:23 — "here is a FLOOR — it refuses the degenerate cases and claims nothing more. The"
  evidence: commands/intent.md:177 — "**Ask for the expectation and its falsifier.** \"Planned it because it is next\""
  evidence: commands/capture.md:139 — "**Ask for the expectation and its falsifier.** \"Promoted it because it is next\""
  evidence: internal/core/lint/schema.go:1264 — "grounds is in frontmatter; a frontmatter scalar is SET, so a later triage route overwrites the conjecture an earlier one recorded"
  evidence: .abcd/development/intents/shipped/itd-179-the-reasoning-behind-what-was-pursued-no-longer-evaporates-a.md:50 — "- pursued: Pursued now because the reasoning behind what went forward is the half of the record that is never written down"
  evidence: .abcd/development/intents/disciplines/itd-192-an-acceptance-criterion-whose-producer-does-not-exist-is-jud.md:27 — "- **Wired in this phase** — the criterion is `MET_WITH_CONCERNS`, and the"

Gap audit:
- honoured:
  - A grounds argument on capture's triage routes, refusing on absence — promote and resolve exit 2 and write nothing, verified by running the shipped binary, with the refusal ordered ahead of the mint so no orphan draft is left behind.
    evidence: internal/core/capture/promote.go:116 — "g, gRedacted, gDegraded, err := requireGrounds(repoRoot, \"promote\", req.Grounds)"
    evidence: internal/core/capture/workflow.go:265 — "g, gRedacted, gDegraded, err := requireGrounds(rr, \"resolve\", req.Grounds)"
  - The readiness gate carries a refusing `grounds` check reported last — exit 1 with a remedy naming the flag and the vocabulary, exit 0 once an entry is recorded. Run, not read.
    evidence: internal/core/intent/ready.go:307 — "c.Detail = \"no recorded grounds — the conjecture behind pursuing \" + it.ID + \" is unrecorded\""
  - The vocabulary is closed and held once as data; the six user-facing spellings render from the definition rather than being typed at each surface.
    evidence: internal/core/grounds/grounds.go:72 — "var Vocabulary = []Token{Pursued, Deferred, Declined}"
    evidence: internal/surface/cli/cli.go:1613 — "Use:   \"ready <itd-N> [--grounds \\\"\" + grounds.UsageSpelling() + \": <conjecture>\\\"]\""
  - Recording is append-only and shared by both record halves through core/mdrecord — a promote then a resolve leaves BOTH bullets on the issue, and two `intent ready` calls leave both on the intent. Verified by running the sequence.
    evidence: internal/core/capture/workflow.go:516 — "newContent, err = appendGrounds(verb, newContent, *g)"
    evidence: .abcd/development/decisions/adrs/0057-grounds-accumulate-as-an-append-only-section.md:13 — "# ADR-57: Grounds accumulate as an append-only section, in one shared form"
  - The substance floor is real and script-aware in the way promised: a 2-word text is refused at the word floor while a Chinese-script text with no inter-word breaks is accepted, so a scriptio-continua script is not refused for having one run.
    evidence: internal/core/grounds/grounds.go:110 — "const MinTextLetters = 20"
    evidence: internal/core/grounds/grounds.go:112 — "// MinTextWords is the same floor measured in LEXICAL UNITS, and it is the half"
  - Wontfix records `declined` from the reason it already takes, needing no new required flag — the record lands with `- declined: <reason>` under `## Grounds`. Verified live.
    evidence: internal/core/capture/grounds.go:80 — "return grounds.Grounds{Token: grounds.Declined, Text: folded}, 0, deg, nil"
  - Grounds live on the intent record and the pre-tooling spec stand-in is retired: zero `## Grounds (pursued)` sections remain on any spec, and thirteen intents carry the section.
    evidence: .abcd/development/specs/closed/spc-57-the-reasoning-behind-what-was-pursued-no-longer-evaporates-a.md:65 — "pre-tooling stand-in was a `## Grounds (pursued)` section on the spec, named as"
  - A frontmatter `grounds:` key is blocked by the record lint and told where the value belongs, closing the overwrite path the section form replaced.
    evidence: internal/core/lint/schema.go:1264 — "grounds is in frontmatter; a frontmatter scalar is SET, so a later triage route overwrites the conjecture an earlier one recorded"
  - Plain `abcd capture` deliberately still takes no grounds, so recording a finding stays cheaper than fixing one — confirmed by capturing an issue with no flags.
    evidence: commands/capture.md:117 — "## Grounds: why this triage, not just which one"
- diverged:
  - 'A grounds argument on the capture triage routes (promote / resolve / wontfix), refusal on absence' is delivered for the ISSUE route only. The rdi route to an intent draft dispatches before the gate, requires no grounds, and delegates to a `disposition_grounds` field that is free text with no vocabulary token and no substance floor — a second grammar under the same flag name on the same verb. The late find fixed before merge (9bdf7d47) closed the worse half of this, where the route ACCEPTED and DISCARDED an operand while exiting 0.
    evidence: internal/core/capture/promote.go:109 — "if reReadingItemID.MatchString(req.ID) {"
    evidence: internal/surface/cli/cli.go:2819 — "dispositionCmd.Flags().StringVar(&dispGrounds, \"grounds\", \"\", \"disposition_grounds: why this answer (free text; required on every state except held)\")"
  - 'The grounds name the conjecture' is delivered as a prose interview prompt plus a degeneracy floor, not as anything that can refuse a restatement of the decision. The spec's own counterexample passes the shipped gate at exit 0 and is written to the record. Disclosed in the spec and in the package doc; still a delta between the criterion's outcome and the delivered mechanism, with no successor intent named to close it.
    evidence: internal/core/grounds/grounds.go:23 — "here is a FLOOR — it refuses the degenerate cases and claims nothing more. The"
    evidence: .abcd/development/specs/closed/spc-57-the-reasoning-behind-what-was-pursued-no-longer-evaporates-a.md:112 — "falsifier. The spec claims the floor, not the judgement."
  - NOT IN THE DECLARED DEVIATIONS. spc-57's staging paragraph reports its own cost wrongly at the state it names. It says 'measured at the branch tip, 10 of the 66 planned/ records carry an entry, 56 fail the grounds check'. Measured at the actual branch tip (build/itd-179 = 9bdf7d47): 63 planned records, 6 carrying a `## Grounds` section, 57 failing. Every one of the three numbers is wrong. This is the exact failure mode iss-2608301918362294 diagnoses — a prose statement of fact about the tree cannot fail, so it cannot be maintained — recurring on a COUNT rather than on an enumeration, inside the record that files the diagnosis.
    evidence: .abcd/development/specs/closed/spc-57-the-reasoning-behind-what-was-pursued-no-longer-evaporates-a.md:129 — "corpus: measured at the branch tip, 10 of the 66 `planned/` records carry an"
    evidence: .abcd/work/issues/open/iss-2608301918362294-a-prose-enumeration-of-where-a-value-is-copied-cannot-be-mai.md:41 — "**A prose enumeration of facts about the codebase cannot fail, so it cannot be"
  - NOT IN THE DECLARED DEVIATIONS, and it understates deviation 3 by an order of magnitude. iss-2608301747006182 says 'fourteen records that reached resolved/ before grounds existed carry none'. The measured figure across the delivered range is far larger: 95 records entered `.abcd/work/issues/resolved/` AFTER the refusal commit e7a3bd5a landed on this branch, and 69 of them carry no `## Grounds` section at all — because they were authored directly into `resolved/` with a `resolution:` frontmatter key rather than passing through `capture resolve`. The gate is not bypassed; it is simply never invoked. That is a wider hole than the record admits, and it means the branch that shipped the obligation is itself the largest producer of grounds-less terminal records.
    evidence: .abcd/work/issues/open/iss-2608301747006182-no-gate-requires-a-terminal-folder-record-to-carry-a-grounds.md:18 — "A gate demanding one would be red today: the fourteen records that reached"
    evidence: .abcd/work/issues/resolved/iss-2608300227224016-record-stores-config-can-hide-a-directory-from-the-gate.md:10 — "resolution: \"The nested-store-root exemption is now derived from the code-side recordStores"
- missing:
  - A record whose captured body leaves an HTML comment or code fence open is locked out of promote, resolve AND wontfix permanently, with no edit/amend/reopen verb to recover it. Reproduced live from ordinary prose with no malice: `abcd capture "...when the < !-- abcd-review marker is stale"` is accepted silently, and all three triage routes then refuse that record forever. The press release promises the triage routes take grounds; for this class of record they take nothing at all. Open as CRITICAL against this intent's own surface, deliberately left live — the delta made it survivable (named construct and line, no orphan draft) and closed the frontmatter half only.
    evidence: .abcd/work/issues/open/iss-2608301908270888-an-unclosed-comment-or-fence-in-an-issue-body-still-locks-th.md:37 — "There is no `edit`, `amend` or `reopen` verb. The only exit is a hand edit of a"
    evidence: internal/core/grounds/record.go:223 — "return \"an HTML comment (`< !--` with no `-- >`)\""
  - Coverage the frontmatter form had and the section form does not: a malformed `grounds:` VALUE was judgeable by the schema gate, but a malformed BULLET under `## Grounds` reads as prose, so the entry is silently absent and no gate says so. Open and unclosed.
    evidence: .abcd/work/issues/open/iss-2608301747001641-a-malformed-grounds-bullet-on-an-issue-record-reads-as-prose.md:19 — "A malformed bullet under `## Grounds` is prose: the reader takes what parses and"
  - No mechanism distinguishes a conjecture from a restatement of the decision, and none is scheduled. spc-57 puts semantic judgement out of scope and names no successor intent; itd-195, the only stub filed on this branch about unverifiable claims, addresses executable claims about code behaviour rather than the judging of grounds prose. Under itd-192 the conditional half of ac-2 has no named closure to be re-checked against at phase close.
    evidence: .abcd/development/specs/closed/spc-57-the-reasoning-behind-what-was-pursued-no-longer-evaporates-a.md:178 — "- Any semantic judgement of whether a grounds text really names a conjecture."
    evidence: .abcd/development/intents/disciplines/itd-192-an-acceptance-criterion-whose-producer-does-not-exist-is-jud.md:27 — "- **Wired in this phase** — the criterion is `MET_WITH_CONCERNS`, and the"
  - The docs still spell the closed vocabulary by hand even though the behavioural copy is now derived — deliberately, and recorded. The behavioural half is genuinely closed (the six user-facing spellings render from `grounds.UsageSpelling()`/`ProseList()`), so what is missing is the executable enumeration the record itself prescribes as the remedy and declines to apply.
    evidence: .abcd/work/issues/open/iss-2608301918362294-a-prose-enumeration-of-where-a-value-is-copied-cannot-be-mai.md:46 — "Remedy, deliberately NOT applied tonight: make the enumeration executable."