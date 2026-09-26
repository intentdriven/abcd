---
schema_version: 1
id: "iss-304"
slug: "confirmed-nitpicks-deferred-from-bughunt-round-1-recorded-no"
severity: "nitpick"
category: "tech-debt"
source: "agent-finding"
found_during: "bughunt-round-1"
found_at: "internal/core/memory/ingest.go"
resolution: "b-4: the generated CLI reference now lists Cobra's completion and help. a-4 was already closed at tip: materialFromLocal reads through fsutil.ReadGuarded, which caps the bytes actually read. d-12 (the record-lint job id) needs a coordinated live ruleset edit and continues as iss-2609251358062952."
impact: fix
resolved_by:
  commit: "cb2c4c95"
---

Confirmed nitpicks deferred from bughunt round 1 (recorded not fixed): (a-4) memory ingest --source reads its operand with os.Stat-then-unbounded os.ReadFile, a size TOCTOU the URL branch's io.LimitReader avoids; (b-4) the generated CLI reference claims to list every user-facing command but omits Cobra's completion/help commands; (d-12) the CI job id 'record-lint' actually runs scripts/check-reviews.sh (reviews-charter), a misleading required-check name whose safe rename needs a coordinated live-ruleset edit
## Evidence

- a-4 — `internal/core/memory/ingest.go` `materialFromLocal` reads a `--source` operand with
  `os.Stat` then unbounded `os.ReadFile`; the size cap is a pre-read stat, so a file grown
  between the two syscalls is read past `maxFetchBytes` (the URL branch bounds with
  `io.LimitReader`). Operator-typed local path → self-inflicted OOM only; nitpick.
- b-4 — `internal/surface/cli/reference.go` generates `docs/reference/cli/commands.md` from
  `NewRootCommand()` before Cobra attaches `completion`/`help`, so the page's "every
  user-facing command" claim omits two invokable commands; nitpick.
- d-12 — `.github/workflows/ci.yml` job id `record-lint` runs `scripts/check-reviews.sh`
  (reviews-charter), while real record-lint is a step of the `check` job; a misleading
  required-check name whose safe rename needs a coordinated live-ruleset edit; nitpick.

## Adversarial review

Each CONFIRMED (nitpick) by an independent refuter. Recorded not fixed this round: a-4/b-4 are
below the fix bar for a substantive round, and d-12's rename cannot be applied without a
coordinated live branch-ruleset change (unsafe autonomously).

## Grounds

- pursued: the reference lists abcd completion and abcd help and a local ingest source is read under the byte cap; a command the binary answers to missing from the reference would show it wrong
