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

> **One user-level home for everything abcd keeps outside your repos — blessed by default, movable by choice.** abcd's repo-tree `.abcd/` tiers cover what belongs to a project. But some state belongs to the *person*: the confidential-sources corpus ([itd-76](../shipped/itd-76-source-provenance-ledger.md)), and whatever user-scoped configuration follows it. That state lives in abcd's user-level home — `~/.abcd/` by default, the same dotfile convention as every tool a developer already trusts with home-directory state. The default is only a default: a guided, wizard-style flow moves the home wherever the user's filing system wants it — say, alongside their projects root — updates the recorded location, and verifies every consumer (guards, skills, verbs) resolves the new path before declaring the move done. The user tier is *additive*: repo `.abcd/` tiers are untouched, and a repo remains fully functional on machines where no user home exists at all.
>
> "My whole development world lives under one directory, and I wanted abcd's user state in there too, not scattered in my home directory," said Alice. "One command moved it, re-pointed everything that reads it, and proved the pre-commit guard still found my banlist before it called the move complete."

## Why This Matters

The moment abcd grew a user-level surface, it inherited a question every long-lived tool answers eventually: whose filing system wins, the tool's or the user's? Hard-coding `~/.abcd/` answers "the tool's" forever; making the path a per-callsite option answers "nobody's" and scatters resolution logic across every consumer. The right shape is a single blessed default plus a single recorded override that every consumer resolves through one function — cheap now, and it prevents the class of bug where one consumer honours the moved home and another silently recreates the old one. A rehearsed, verified move matters more than the move itself: user state includes material whose guards must not lapse mid-relocation (itd-76's banlist source), so the flow proves consumers resolve the new location before the old one is retired.

## What It Looks Like

- **One resolution rule.** Every abcd component that touches user-level state resolves the home through a single lookup: explicit config, else environment override, else `~/.abcd/`. No consumer hard-codes the path.
- **A guided move.** An interactive, wizard-style flow relocates the home: copies (never moves-then-prays) the tree to the target, records the new location, re-runs each consumer's own verification (the itd-76 guard finding its banlist source, skills resolving the corpus), and only then retires the old tree — a rehearsed cutover with a rollback path, not a rename.
- **Additive to repo tiers.** The user home complements the repo's `.abcd/` layout; nothing repo-side moves, and every repo continues to work on machines with no user home (guards no-op exactly as the itd-74 banlist does on fresh clones).
- **Doctor-visible.** The install-health check reports where the user home resolves and flags a dangling override (recorded path that no longer exists).

## Open Questions

- Where the override is recorded: a user-level config file at a fixed bootstrap location (a pointer must live *somewhere* unmovable), an environment variable, or both with defined precedence.
- Whether repo-level config may pin a user-home path for reproducibility, or whether that inverts the tiers (a repo dictating personal filing) and should be refused.
- Migration of stale references after a move — e.g. per-repo banlists regenerated on next commit by the itd-76 auto-refresh, or swept eagerly by the move flow.
