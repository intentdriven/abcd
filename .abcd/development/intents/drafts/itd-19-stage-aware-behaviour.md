---
id: itd-19
slug: stage-aware-behaviour
spec_id: null
kind: standalone
suggested_kind: null
reclassification_history: []
severity: minor
---

# Defaults That Match Where the Project Lives

## Press Release

> **abcd detects the development stage of a project (Planning / Autonomous / Collaborative / Validation) and adjusts its defaults accordingly.** A project under `Autonomous/` runs ahoy with permissive defaults (private visibility, all dev-sync sources enabled, deep secret scanning on); the same project moved to `Validation/` runs with restrictive defaults (read-mostly, dev-sync disabled, all gates strict). Stage-aware behaviour means abcd does the right thing per-stage without users having to reconfigure.
>
> "I move projects between stages as they mature," said Grace, VP Engineering. "abcd detected the stage, asked me whether to apply stage defaults, and tightened up automatically when I moved to Validation. One less thing to remember."

## Why This Matters

abcd v0 had stage detection via chezmoi (`~/.claude/commands/abcd.md` reads `chezmoi data | jq .devPath`) but applied it only at command-dispatch time. abcd doesn't do stage detection at all — every project has the same defaults regardless of where it lives.

The ABCDevelopment workflow has stages with genuinely different intent (Autonomous = AI runs free, Validation = read-mostly QA). abcd should respect that. This intent makes stage detection optional (not every user uses ABCDevelopment) but properly integrated when present.

**Note:** specific to users adopting the ABCDevelopment stage convention. Generic abcd users (no chezmoi, no ABCDevelopment) get the default profile unchanged.

## What's In Scope

- Stage detection via chezmoi (`chezmoi data --format json | jq -r '.devPath'`) when chezmoi is present
- Per-stage default profiles for `.abcd/config.json`:
  - Autonomous: visibility=private, dev_sync=all-on, scan.deep=true, oracle=in-session (fast iteration)
  - Collaborative: visibility=private, dev_sync=all-on, scan.deep=true, oracle=rp (cross-model perspective)
  - Validation: visibility=public-prep, dev_sync=read-only, scan.deep=true, oracle=rp+codex (max scrutiny)
  - Planning: visibility=public-prep, dev_sync=all-on, scan.deep=false (still designing)
- ahoy detects stage, transparent prompt: "Apply stage defaults? Currently in Autonomous; recommended config: ..."
- Stage transitions (project moves between stages) trigger a re-confirm at next ahoy

## What's Out of Scope

- Forcing stage-specific behaviour without user opt-in (abcd is opinionated but not coercive)
- Detecting non-chezmoi stage conventions (out of scope until a generalisable pattern emerges)
- Stage-specific commands (e.g., `/abcd:validate-mode`) — config defaults are sufficient

## Acceptance Criteria

> _BDD format, per `itd-1-acceptance-gates`. These gates are checked by `intent-fidelity-reviewer` when this intent moves to `shipped/`._

- **Given** a project under `~/ABCDevelopment/Autonomous/<name>/` with chezmoi configured, **when** `/abcd:ahoy install` runs, **then** stage detection identifies "Autonomous", proposes the Autonomous default profile (visibility=private, dev_sync=all-on, scan.deep=true, oracle=in-session), and asks for explicit confirmation before applying.
- **Given** a non-chezmoi user (or a project outside ABCDevelopment), **when** `/abcd:ahoy install` runs, **then** stage detection returns "unknown" AND the user receives the default profile unchanged — no warning, no prompt about ABCDevelopment.
- **Given** a project moves from `Autonomous/<name>/` to `Validation/<name>/`, **when** the user re-runs `/abcd:ahoy install` in the new location, **then** stage detection identifies "Validation", proposes the Validation profile (visibility=public-prep, dev_sync=read-only, scan.deep=true, oracle=rp+codex), and surfaces a diff of what would change vs. the current `.abcd/config.json`.
- **Given** the user has explicitly overridden a config setting (e.g., set `dev_sync.rp.enabled = false` in Autonomous), **when** the stage transitions and the new profile would re-enable that setting, **then** the user is asked specifically about that override (preserve / re-prompt-each / accept-new-default) rather than silently overwriting.
- **Given** a Validation-stage project with `oracle.backend = "rp+codex"` requested by the profile, **when** RP is unavailable, **then** the project gracefully falls back per the resolution chain AND the disembark report flags "stage profile requested rp+codex but only Codex available".
- **Given** a stage-aware project, **when** `/abcd:disembark pack <repo> <path>` runs, **then** the resulting lifeboat's `.abcd/config.json["meta"]` records the stage detected at disembark time so a downstream `/abcd:embark from <path>` can suggest the matching stage profile if its target location indicates the same stage.

## Open Questions

- What if the user explicitly overrode a config setting that the new stage profile would change — preserve override or re-prompt?
- Is the Validation profile's "read-mostly" implemented via permission tightening, or is that a separate concern?
- Should stage-aware behaviour be exposed in the lifeboat (so an embarked project knows what stage it expects to live in)?

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._
