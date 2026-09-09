---
schema_version: 1
id: "iss-33"
slug: "ahoy-verb-hygiene"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "2026-07-08 multi-agent review"
found_at: "internal/core/ahoy/apply.go"
---

RE-SCOPED 2026-09-09 against the shipped tree. Three of the four instances in the
original acceptance corpus were fixed elsewhere and are struck below; two things
survive, and only one of them is what the record was mostly about.

SURVIVING — `ahoy.Status` is silent dead scaffolding. `internal/core/ahoy/apply.go:1352`
declares `func Status(cwd string) (string, error)`, the bare-command human summary,
and NOTHING calls it: not the CLI, not the plugin surface, not a test. The
original record said "zero callers outside tests"; the true state is stronger —
there are no callers at all, so the function is not even pinned by a test that
would notice it rotting. It is not merely unwired: `01-ahoy.md` lists `status`
among the sub-verbs that ship on the CLI while `abcd ahoy status` exits 2, which
`iss-2609020734198139` records separately as a false claim in the design record.
Either the verb gets a front door or the function and the chapter line go; what
is not tenable is an exported renderer nothing can reach.

SURVIVING (narrower than recorded) — the `scan_deep` answer is still coerced,
on the collect-missing path only. `apply.go:490` reads
`v := a.resolveValue("scan_deep", []string{"true", "false"}, "false") == "true"`,
so any answer that is not the literal `true` becomes `false` with no diagnostic.
`resolveValue` (`apply.go:531`) hands the choice set to the prompter but does not
enforce it, and `stdinPrompter.Prompt` (`internal/surface/cli/cli.go:2462`)
returns the typed line verbatim — so a user who answers `yes` at the
`scan_deep (true/false) [false]:` prompt gets the deep secret scanner switched
OFF, having said to switch it on. The asymmetry is the point: the three
neighbouring slots re-validate what the prompter returned and abandon the install
rather than persist a typo — `apply.go:473`, `:479`, `:485` each test
`inSet(value, choices)` and `return nil`, the partial-install path. `scan_deep`
alone skips that check. The override path is already validated
(`applyScanDeepOverride`, `apply.go:591`, ignores anything but `true`/`false`),
so the gap is the interactive/collect-missing route.

FIXED ELSEWHERE, struck from the corpus:

- The `docs_target` and `oracle_backend` halves of the persist-unvalidated claim.
  6cf0df8a (2026-07-15) added the `inSet` re-validation; `apply.go:478-481` now
  carries the comment `// no valid docs target => partial (never persist a typo)`
  and `:484-487` the same for the oracle backend. Neither slot can persist an
  answer outside its choice set.
- `registerRepo` is neither untested nor silent. `internal/core/ahoy/lockrace_test.go`
  drives it directly through two concurrent registrations, and c88c94b4
  (2026-08-28) replaced the swallowed error with a change-note: `apply.go:825`
  appends `history registration for <sha> skipped (<err>)` on a lock failure, and
  the lineage-conflict branch below it reports its own refusal (iss-128).
- `doctor` and `dry-run` have behavioural tests against a real temporary repo.
  `internal/surface/cli/cli_test.go:883`, `:912`, `:962` and `:1011` exercise
  `abcd ahoy doctor` in both JSON and text renders;
  `internal/surface/cli/ahoy_banlist_reach_test.go:102` exercises
  `abcd ahoy dry-run`.

Every line number in the ORIGINAL text was wrong for the current tree (it cited
`apply.go:184-192`, `:280`, `:441`, `:478`); the file has roughly tripled since.
The detector the record asked for — a wired-verb convention plus a
coverage-plus-caller audit that distinguishes loud staging from silent
scaffolding, per `.abcd/development/principles/loud-staging.md` — is still the
right shape and is still unbuilt; `ahoy.Status` is exactly the instance it would
catch.
