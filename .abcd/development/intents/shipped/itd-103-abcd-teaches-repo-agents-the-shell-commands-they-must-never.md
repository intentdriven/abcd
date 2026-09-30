---
id: itd-103
slug: abcd-teaches-repo-agents-the-shell-commands-they-must-never
spec_id: spc-16
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
related_adrs: [adr-42]
---

# **abcd teaches repo agents the shell commands they must never run — and blocks them when they try.** An agent that runs `cd scratch && rm -rf *` is one failed `cd` away from deleting the working tree, and the facilitator watching the session usually has no way to know that. abcd ships a hazard registry: each entry a dangerous command pattern, its severity, the safe successor, and a plain-language why. One registry, two planes. The rules loader injects the matched safety rules before shell-heavy work, so the agent is taught the safe form up front; a deterministic guard — a core verb any harness hook can call — refuses a matching command at execution time and replies with the safe successor, so the block itself is the lesson. Hosts without hook support still get the teaching plane; hosts with hooks get both, wired in at install. The registry grows from reality: a facilitator who sees something scary needs to know exactly one move — capture it — and recurring captures promote patterns into the bundled defaults, each entry shipping with a fixture that proves its guard fires. "I couldn't have told you why that command was dangerous," said Nia, a facilitator. "I didn't have to. The guard refused it, told the agent what to run instead, and told me why in words I understood. The one time something new scared me, I captured it and moved on."

## Press Release

> **abcd teaches repository agents which shell commands are dangerous — and blocks them at the moment one is about to run.** abcd ships a hazard registry in which each entry pairs a dangerous command pattern with its severity, its safe successor, and a plain-language reason. One registry drives two planes. Before shell-heavy work the rules loader injects the matching safety rules, so the agent learns the safe form up front; at execution time a deterministic guard — a core verb any harness hook can call — refuses a matching command and answers with the safe successor, so the block itself is the lesson. A host without hook support still gets the teaching plane; a host with hooks gets both, wired in at install. The registry grows from reality: a facilitator who sees something alarming has exactly one move — capture it — and recurring captures promote patterns into the bundled defaults, each entry shipping with a fixture that proves its guard fires.

## Why This Matters

**abcd teaches repo agents the shell commands they must never run — and blocks them when they try.** An agent that runs `cd scratch && rm -rf *` is one failed `cd` away from deleting the working tree, and the facilitator watching the session usually has no way to know that. abcd ships a hazard registry: each entry a dangerous command pattern, its severity, the safe successor, and a plain-language why. One registry, two planes. The rules loader injects the matched safety rules before shell-heavy work, so the agent is taught the safe form up front; a deterministic guard — a core verb any harness hook can call — refuses a matching command at execution time and replies with the safe successor, so the block itself is the lesson. Hosts without hook support still get the teaching plane; hosts with hooks get both, wired in at install. The registry grows from reality: a facilitator who sees something scary needs to know exactly one move — capture it — and recurring captures promote patterns into the bundled defaults, each entry shipping with a fixture that proves its guard fires. "I couldn't have told you why that command was dangerous," said Nia, a facilitator. "I didn't have to. The guard refused it, told the agent what to run instead, and told me why in words I understood. The one time something new scared me, I captured it and moved on."

## Acceptance Criteria

- Given the guard hook cannot execute the abcd binary, when any command runs, then the guard fails OPEN with an unmissable in-session warning, and guard health is reported by ahoy status — never a silent no-op, never a bricked session.
- Given a command matches a blocker-tier registry entry, then it is refused with the safe successor as the block message; warn-tier matches pass with the warning injected; no in-session override exists — the only escape is a committed per-repo config override.
- Given a hazard pattern appears inside a quoted string argument (for example a ledger capture whose text mentions a dangerous command), when the guard evaluates, then it does not fire: matching is shell-token-aware and applies in command position only, including cd-chain structure across compound separators.
- Given a registry entry is proposed for the bundled defaults, then it ships with known-bad and known-good fixtures (known-good at least 40% of its corpus) and clears a declared true-negative-rate floor before admission; string payloads inside eval or shell -c are a documented v1 gap, not a silent one.

## Open Questions

- The registry config's file home (extend rules.json with a guard domain vs a dedicated committed file) — a spec-time decision.
- Which non-shell tool calls (if any) the guard family later covers.

## Grill Settlements (2026-07-27)

- Fail-open-loud on guard breakage; blocker/warn tiers mirror the docs-lint family; overrides are committed and reviewable only.
- Shell-token-aware matching in command position was chosen precisely because a raw-regex guard would have blocked this repo's own incident-capture command — the known-good corpus is not optional.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-c09c73d6adc5 -->
Fidelity review — receipt rcp-c09c73d6adc5 (verifier abcd:intent-fidelity-reviewer claude-fable-5).

Provenance: abcd:intent-fidelity-reviewer@claude-fable-5 · rubric_hash sha256:e96fe6d7a3f6bfe3908171b52f5bd42de194298ead0093ffab6b377adc9dd3bc · prompt_hash sha256:f4de8b23c6b6c7a54133bf2042b6b89829c579412cf26f1f66a65c36d57aa6fe
Input attestations: diff:main..feat/itd-103-shell-hazard-registry (PR #156, merge 49cffa05): internal/core/guard/, internal/surface/cli/guard.go, internal/core/ahoy/guard_health.go, commands/abcd/guard.md, hooks/hooks.json@sha256:49cffa05e6216d9cbb29dc601961464781e6541d; rubric:.abcd/.work.local/reviews/rcp-c09c73d6adc5.request.md@sha256:e96fe6d7a3f6bfe3908171b52f5bd42de194298ead0093ffab6b377adc9dd3bc;

Acceptance rollup: MET 3 · MET_WITH_CONCERNS 1 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: Every non-decision path in the hook adapter routes through one failOpen that exits 1 (non-zero, non-blocking) and prints 'NOT CHECKED — ... runs UNGUARDED'; the hooks.json wrapper catches even an unexpected exit and fails open loudly; and guard health is a first-class ahoy field wired into the status render — never a silent no-op, never a bricked session.
  evidence: internal/surface/cli/guard.go:145 — "\"abcd guard: NOT CHECKED — \"+format+\". This command runs UNGUARDED.\\n\" ... return &exitError{Code: 1}"
  evidence: hooks/hooks.json:13 — "[ $s -eq 0 ] || [ $s -eq 1 ] || [ $s -eq 2 ] || { echo \"abcd guard: FAILED TO RUN (exit $s) — shell commands run UNGUARDED ...\" >&2; exit 1; }"
  evidence: internal/core/ahoy/detect.go:78 — "res.Guard = detectGuardHealth(abs, pluginRoot, pluginOK)"
  evidence: internal/surface/cli/cli.go:1453 — "fmt.Fprintf(w, \"  guard:       %s\\n\", guardHealthLine(res.Guard))"
- ac-2 — MET_WITH_CONCERNS: A blocker refuses with the successor built into the block message and a warn passes with the warning surfaced, and the only disable route is the per-repo .abcd/guard.json config (no in-session flag or env bypass exists) — CONCERN: that override takes effect from the WORKING TREE before it is committed or reviewed (config.Load reads .abcd/guard.json off disk; iss-147 records this exactly), so it is reviewable-in-a-diff but the criterion's 'committed' qualifier is not actually enforced.
  evidence: internal/core/guard/guard.go:300 — "return fmt.Sprintf(\"%s by the abcd guard (%s): %s Run instead: %s\", lead, e.ID, e.Why, e.Successor)"
  evidence: internal/surface/cli/guard.go:198 — "case guard.VerdictBlock: ... fmt.Fprintln(cmd.ErrOrStderr(), termsafe.Sanitize(dec.Message)); return &exitError{Code: 2}"
  evidence: internal/core/guard/config.go:34 — "data, err := fsutil.ReadGuarded(filepath.Join(repoRoot, \".abcd\", \"guard.json\"), maxGuardFileBytes)"
  evidence: .abcd/work/issues/open/iss-147-guard-load-reads-abcd-guard-json-from-the-working-tree-so-a.md:4 — "guard-load-reads-abcd-guard-json-from-the-working-tree"
- ac-3 — MET: tokenize honours shell quoting and recognises operators outside quotes only, so a hazard inside a quoted argument stays one non-command token; matching runs in command position via commandOf/matchSegment; the cd-chain is enforced by AfterCD + precededByCD scanning earlier segments of the SAME chain across &&/;/||/| separators; and TestIncidentCaptureIsKnownGoodFixtureOne pins this repo's own quoted 'cd scratch && rm -rf *' capture as a known-good that must not fire.
  evidence: internal/core/guard/tokenize.go:23 — "Operators and comments are recognised OUTSIDE quotes only, which is exactly what keeps a hazard named inside a quoted argument from ever reaching command position."
  evidence: internal/core/guard/match.go:41 — "if matchSegment(p, s) && (p.AfterCD == nil || !*p.AfterCD || precededByCD(segs[:i], s.chain))"
  evidence: internal/core/guard/match.go:51 — "func precededByCD(before []segment, chain int) bool { ... if cmd, _ := commandOf(s); cmd == \"cd\""
  evidence: internal/core/guard/fixtures_test.go:104 — "if !containsAll(first, \"capture\", \"cd scratch && rm -rf *\")"
- ac-4 — MET: The admission gate TestBundledEntriesPassAdmissionGate fails the build unless every bundled entry ships known-bad AND known-good fixtures, known-good is at least 40% of the corpus (10*good < 4*(good+bad)), and every known-good fires nothing (a declared 100% true-negative floor); the 6 shipped entries all clear it (e.g. rm-rf-after-cd-chain 8/17 good = 47%); and the eval / sh -c string-payload gap is documented, not silent, in the tokenizer, the CLI reference, and the surface brief.
  evidence: internal/core/guard/fixtures_test.go:41 — "if 10*good < 4*(good+bad) { t.Fatalf(\"entry %s: known-good is %d of %d fixtures, below the 40%% floor\" ...)"
  evidence: internal/core/guard/fixtures_test.go:66 — "if d.Verdict != VerdictAllow { t.Fatalf(\"known-good %q fired %v ...; the true-negative floor is 100%%\" ...)"
  evidence: internal/core/guard/tokenize.go:26 — "v1 GAP (documented, not silent): ... A command string carried as a DATA argument — `sh -c '<payload>'`, `eval '<payload>'` ... its payload is never parsed"
  evidence: .abcd/development/brief/04-surfaces/17-guard.md:86 — "a command string handed to an interpreter (`eval`, `sh -c`)"

Gap audit:
- honoured:
  - A deterministic guard, a core verb any harness hook can call, refuses a matching command and replies with the safe successor so the block itself is the lesson
    evidence: internal/core/guard/guard.go:246 — "func (r Registry) Check(command string) (Decision, error)"
    evidence: internal/surface/cli/guard.go:122 — "newGuardHookCommand builds `guard hook`: the adapter between a host's pre-tool-use hook payload and the core decision"
  - Two tiers mirroring the docs-lint family: a blocker refuses, a warn passes with the warning attached
    evidence: internal/core/guard/guard.go:19 — "TierBlocker = \"blocker\" ... TierWarn = \"warn\""
    evidence: internal/surface/cli/guard.go:200 — "case guard.VerdictWarn: fmt.Fprintln(cmd.ErrOrStderr(), termsafe.Sanitize(dec.Message)); return nil"
  - Each entry ships a fixture that proves its guard fires, enforced as a build-time admission gate
    evidence: internal/core/guard/fixtures_test.go:19 — "func TestBundledEntriesPassAdmissionGate"
    evidence: internal/core/guard/defaults/guard.json:1 — "6 entries, each carrying known_bad + known_good fixtures"
  - The known-good corpus is not optional: this repo's own incident-capture command must not be blocked
    evidence: internal/core/guard/fixtures_test.go:95 — "func TestIncidentCaptureIsKnownGoodFixtureOne"
- diverged:
  - The only escape is a COMMITTED per-repo config override — delivered as a per-repo .abcd/guard.json read from the working tree, so it activates before commit/review (reviewable in a diff, but 'committed' is not enforced)
    evidence: internal/core/guard/config.go:34 — "fsutil.ReadGuarded(filepath.Join(repoRoot, \".abcd\", \"guard.json\") ...)"
    evidence: internal/surface/cli/guard.go:275 — "the file is read from the working tree, so this can be true before anyone has reviewed the edit that made it true (iss-147)"
- missing:
  - Press-release 'two planes': the teaching plane — the rules loader injecting the matched safety rules before shell-heavy work — is not wired in this delivery; only the execution-time guard plane ships (a guard/safety rules domain does not exist in internal/core/rules/)
    evidence: .abcd/development/intents/shipped/itd-103-abcd-teaches-repo-agents-the-shell-commands-they-must-never.md:13 — "The rules loader injects the matched safety rules before shell-heavy work"
    evidence: internal/core/rules/rules.go:401 — "no guard/safety/hazard domain is registered — the only `guard` here is the stemming short-token guard"

Changed on 2026-09-29 by the product thinker's ruling J10 of that day (the DECISIONS.md entry landing with this change), recorded as iss-151: the teaching plane the headline promises, missing from the delivery reviewed above, is built. The rules loader bundles a `SHELL` domain generated from the same bundled hazard registry the guard reads, one rule per entry (the command, whether the guard refuses or warns, the why and the safe successor) recalled by the registry's command heads, so a prompt about shell-heavy work is taught the safe form before a command runs, on a host without hooks as on one with them. The criteria above stand as shipped; none of them concerned the teaching plane, and no ADR's text changes.
