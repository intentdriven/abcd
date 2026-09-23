---
id: spc-2609230613208843
slug: per-platform-staleness-calibration
intent: itd-111
origin: researcher-authored
production_mode: hand-written
---
# Staleness behaviour is calibrated per platform

## Summary

The remainder of [itd-111](../../intents/planned/itd-111-a-stale-abcd-never-answers-silently-every-surface-that-runs.md)
that [spc-22](../closed/spc-22-a-stale-abcd-never-answers-silently-every-surface-that-runs.md) did not deliver. spc-22 closed on
2026-09-23 with acceptance criteria 1 to 7 delivered: the SessionStart staleness notice, the install refusal on a stale or unknown vintage, install mode and vintage on `version` and `ahoy`, no version-discovery network request without `version --check`, the transition report, and the unknown-never-fresh outcome. This spec carries what did not ship.

The delivered part was already announced in the [0.5.0] changelog section,
and the intent carries no `shipped_in:` because it stays planned. The close
that ships the intent through this spec is the one that decides how the cut
reports it: without `shipped_in:` the whole intent is announced again, which
is the redundant line the changelog composer prefers to a silent omission.

## Scope

- **Acceptance criterion 8 (missing).** Given the same staleness scenarios on macOS and on Linux, when the itd-109 calibration runs, then observed behaviour matches, recorded as a machine-class criterion, human-verified per platform. No recorded macOS or Linux human-verified calibration run exists on `main` at closure.
