---
id: itd-77
slug: relocatable-user-home
spec_id: null
kind: standalone
suggested_kind: null
reclassification_history: []
severity: minor
blocked_by: []
builds_on: []
---

# abcd's User-Level Home Lives Where You Want It

## Press Release

> **One user-level home for everything abcd keeps outside your repos — blessed by default, movable by choice.** abcd's repo-tree `.abcd/` tiers cover what belongs to a project. But some state belongs to the *person*: the confidential-sources corpus ([itd-76](../shipped/itd-76-source-provenance-ledger.md)), and whatever user-scoped configuration follows it. That state lives in abcd's user-level home — `~/.abcd.noindex/` by default, the same dotfile convention as every tool a developer already trusts with home-directory state. The default is only a default: a guided, wizard-style flow moves the home wherever the user's filing system wants it — say, alongside their projects root — updates the recorded location, and verifies every consumer (guards, skills, verbs) resolves the new path before declaring the move done. The user tier is *additive*: repo `.abcd/` tiers are untouched, and a repo remains fully functional on machines where no user home exists at all.
>
> "My whole development world lives under one directory, and I wanted abcd's user state in there too, not scattered in my home directory," said Alice. "One command moved it, re-pointed everything that reads it, and proved the pre-commit guard still found my banlist before it called the move complete."

## Why This Matters

The moment abcd grew a user-level surface, it inherited a question every long-lived tool answers eventually: whose filing system wins, the tool's or the user's? Hard-coding `~/.abcd.noindex/` answers "the tool's" forever; making the path a per-callsite option answers "nobody's" and scatters resolution logic across every consumer. The right shape is a single blessed default plus a single recorded override that every consumer resolves through one function — cheap now, and it prevents the class of bug where one consumer honours the moved home and another silently recreates the old one. A rehearsed, verified move matters more than the move itself: user state includes material whose guards must not lapse mid-relocation (itd-76's banlist source), so the flow proves consumers resolve the new location before the old one is retired.

## What It Looks Like

- **One resolution rule.** Every abcd component that touches user-level state resolves the home through a single lookup: explicit config, else environment override, else `~/.abcd.noindex/`. No consumer hard-codes the path.
- **A guided move.** An interactive, wizard-style flow relocates the home: copies (never moves-then-prays) the tree to the target, records the new location, re-runs each consumer's own verification (the itd-76 guard finding its banlist source, skills resolving the corpus), and only then retires the old tree — a rehearsed cutover with a rollback path, not a rename.
- **Additive to repo tiers.** The user home complements the repo's `.abcd/` layout; nothing repo-side moves, and every repo continues to work on machines with no user home (guards no-op exactly as the itd-74 banlist does on fresh clones).
- **Doctor-visible.** The install-health check reports where the user home resolves and flags a dangling override (recorded path that no longer exists).

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in
the [install-update-release SOTA note](../../research/notes/2026-10-23-install-update-release-sota.md).

- **Every surveyed tool relocates through an environment variable; none
  documents a move command or a pointer file**: `RUSTUP_HOME` and
  `CARGO_HOME` (rustup), `MISE_DATA_DIR`, `MISE_CONFIG_DIR`, `MISE_CACHE_DIR`
  and `MISE_STATE_DIR` with XDG fallbacks (mise, which warns to set them "in
  the environment that starts mise", never in the tool's own config, or "one
  command [may] use a different directory than the next"), `GH_CONFIG_DIR`
  then `$XDG_CONFIG_HOME/gh` then `$AppData/GitHub CLI` then `~/.config/gh`
  (gh), and the harness's own configuration-directory variable.
- **XDG names what a home holds**: data (`~/.local/share`), config
  (`~/.config`), state (`~/.local/state`: logs, history, "state... that can be
  reused on a restart") and cache (`~/.cache`, "non-essential"); all paths
  "must be absolute" and a relative one is "invalid" and ignored. Go's
  `os.UserConfigDir` and `os.UserCacheDir` apply this on Unix, return
  `~/Library/...` on macOS and `%AppData%`/`%LocalAppData%` on Windows, and
  error on a relative value.
- **macOS's own guidance is `~/Library/Application Support/<bundle id>`**, and
  of the tools surveyed only mise follows it, and only for its cache; rustup,
  uv, gh and the harness keep dot-directories or `~/.local` there.
- **No tool lets a project pin a person's home**: the override is always in
  the person's environment or settings.
- **The one-resolver half already exists**: `internal/abcdhome` is the one
  place that spells the home, held by test; the override and the move are what
  is missing.

**Path 2 — native, shaped to the XDG and `GH_CONFIG_DIR` precedence, with the
resolver as the seam**: the variable first (the field's consensus), a pointer
file at a fixed location second (what no tool has, but what a move abcd
verifies needs, since it cannot assume the person exported anything), the
default third; relative values refused with the reason; a repo-level pin
refused (open question 2); no eager reference sweep on move, since no
surveyed tool does one and the per-repo banlist refresh covers it (open
question 3). No new dependency. The independent fit-challenge runs at plan
time.

## Open Questions

- (SOTA 2026-10-23) What is the fixed location of the pointer file that the
  variable cannot move: `os.UserConfigDir()/abcd`, or a file inside the
  default home that is read only when the home has not moved?
- (SOTA 2026-10-23) Does the symlinked-home refusal (a dotfiles checkout
  cannot host `~/.abcd.noindex`) apply to the moved location, so the move is
  not the way round the rule?
- Where the override is recorded: a user-level config file at a fixed bootstrap location (a pointer must live *somewhere* unmovable), an environment variable, or both with defined precedence.
- Whether repo-level config may pin a user-home path for reproducibility, or whether that inverts the tiers (a repo dictating personal filing) and should be refused.
- Migration of stale references after a move — e.g. per-repo banlists regenerated on next commit by the itd-76 auto-refresh, or swept eagerly by the move flow.
