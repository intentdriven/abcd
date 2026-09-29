---
id: itd-13
slug: scheduled-dev-sync
spec_id: null
kind: standalone
suggested_kind: null
reclassification_history: []
severity: minor
---

# Reviews and Memory Stay Fresh, Always

## Press Release

> **abcd keeps `.abcd/work/` in sync with its volatile sources continuously.** A scheduled launchd agent (or cron entry) runs `abcd dev-sync` every hour by default, promoting new chats from whichever oracle adapter is configured into `.abcd/work/reviews/`, distilling new Claude memories into `.abcd/memory/`, and curating new entries into the `.abcd/work/issues/` ledger. Each source is adapter-scoped: dev-sync pulls only from the sources that are actually configured. By the time you run `/abcd:disembark`, everything is already current.
>
> "Sync ran only at disembark, which meant my lifeboats were always missing the last few hours of oracle reviews," said Frank, SRE. "Now it syncs in the background. When I disembark, I know what's in `.abcd/work/` is what's in my recent work."

## Why This Matters

abcd deliberately scopes `dev-sync` to two triggers: implicit at disembark Phase 0, and manual `abcd dev-sync`. Live/scheduled sync was deferred to keep the initial surface area small and avoid the operational complexity of installing system-level scheduled jobs.

Once real usage patterns emerge, the cost of "stale `.abcd/work/` until next disembark" will become visible. This intent closes that gap with a real scheduled-sync solution.

## What's In Scope

- launchd agent (macOS) and cron entry (Linux) installation via `/abcd:ahoy`
- Configurable interval (default 1h, settable via `.abcd/config.json` → `dev_sync.interval`)
- Per-source enable/disable (`dev_sync.{oracle,memory,work}.enabled`), where the oracle source is adapter-scoped — it activates only when an oracle adapter is configured; no source is mandatory
- Lifecycle: install during ahoy with transparent confirm; uninstall removes the scheduled job
- Health check: `abcd dev-sync status` shows last successful run timestamp

## What's Out of Scope

- Real-time fsevents-based sync (too invasive; cron/launchd is sufficient)
- Cross-machine sync (oracle-adapter chats from one machine appearing in another's `.abcd/work/`)
- Pull-based sync from sources beyond the current set (configured oracle adapters, memory, .abcd/.work.local)

## Acceptance Criteria

> _BDD format, per `itd-1-acceptance-gates`. These gates are checked by `intent-fidelity-reviewer` when this intent moves to `shipped/`._

- **Given** a macOS user runs `/abcd:ahoy` on a fresh repo with `dev_sync.scheduled.enabled = true` in their config, **when** ahoy completes, **then** a launchd `~/Library/LaunchAgents/dev.abcd.devsync.<repo-id>.plist` is installed (with explicit confirmation prompt) AND the user can verify it via `launchctl list | grep abcd`.
- **Given** a Linux user runs `/abcd:ahoy` on the same config, **when** ahoy completes, **then** a cron entry is installed (with explicit confirmation) AND the user can verify it via `crontab -l`.
- **Given** an installed scheduled `dev-sync` runs at its configured interval, **when** the run completes successfully, **then** it writes `.abcd/logbook/dev-sync/<utc-ts>/run-report.{json,md}` AND `abcd dev-sync status` reports the timestamp of the last successful run.
- **Given** a scheduled `dev-sync` run conflicts with an in-progress `/abcd:disembark pack <repo> <path>` (file-system contention on `.abcd/work/`), **when** the scheduler fires, **then** the scheduled run detects the active disembark via a documented lock mechanism, skips its work, logs the skip-reason to its run report, and waits for the next interval.
- **Given** the user runs `/abcd:ahoy uninstall`, **when** the sub-verb completes, **then** the launchd plist (or cron entry) is removed AND the user can verify its absence via the same `launchctl list` / `crontab -l` check.
- **Given** the user changes `dev_sync.interval` in `.abcd/config.json` from `"1h"` to `"30m"`, **when** the next `/abcd:ahoy install` (idempotent re-install) runs, **then** the scheduled job's interval updates to 30 minutes AND the change is recorded in the install report.
- **Given** a scheduled `dev-sync` run fails (a configured oracle adapter unreachable, memory lock contention, etc.), **when** the next scheduled fire occurs, **then** the failure is recorded in the run report, surfaced via `abcd dev-sync status` as "last run: failed (N intervals ago)", and the next run attempts independently — no exponential backoff that hides recurring failures.

## Open Questions

- launchd vs cron — install both with platform detection?
- Should the scheduled job log to `.abcd/logbook/dev-sync/<timestamp>/` (consistent with command logs) or a dedicated location?
- What happens if a scheduled run conflicts with an in-progress disembark — locking, queuing, or skip?

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._
