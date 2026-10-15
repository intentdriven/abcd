---
schema_version: 1
id: "iss-2610150526192138"
slug: "derived-values-and-append-at-end-lists"
severity: "minor"
category: "tech-debt"
source: "agent-observation"
found_during: "merge-hotspots worktree, 2026-10-15, measuring conflicts across the last three weeks of surface-adding PRs"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/release-gate/manifest.json"
remedy: "Store no derived value a surface addition changes: the example receipt carries a placeholder manifestHash that receipt_gate refuses, checkerCount is computed from briefDocs and surfaces, the rosters move out of the hashed prompt text into sorted one-per-line arrays, and briefDocs is ordered by chapter slug, each order and equality held by TestReleaseGateManifestIsCurrent."
---

Derived values and append-at-end lists in the release-gate manifest and surface registers conflict on every surface-adding branch. Every branch that adds a command page, agent prompt or surface chapter rewrites the same lines of .abcd/development/release-gate/manifest.json (checkerCount, promptHash, and the one-line prompt context that carries the command and agent rosters) and the manifestHash literal in receipt.example.json, which is the sha256 of the manifest, so two such branches always conflict and the second must merge main and run a fresh preflight before it can queue again.

## Measurement

The ten PRs merged into main between 2026-09-24 and 2026-10-15 whose branch
added a command page, an agent prompt or a brief surface chapter: #725 (lab),
#727 (scribe), #729 (build, source), #752 (drain), #762 (reflect,
reflection-composer), #814 (planning-interviewer), #828 (dashboard), #849
(question-drafter), #922 (library, a rename of memory/source) and #1004
(reserve). Counts are of those ten branches (`git diff <merge>^1...<merge>^2`)
changing the named line or value.

| File | Line or value | Derived or authored | Touched | Generated or hand-kept |
|---|---|---|---|---|
| `.abcd/development/release-gate/manifest.json` | `prompt.context`, one JSON line carrying the whole command and agent roster | roster derived (held equal to `commands/` and `agents/`) | 10/10 | hand-kept |
| same | `promptHash` | derived: sha256 of context, directionA, directionB | 10/10 | hand-kept |
| same | `checkerCount` | derived: len(briefDocs) + len(surfaces) | 8/10 | hand-kept |
| same | `briefDocs`, the line after the last `04-surfaces/NN-*.md` entry | derived (every 04-surfaces chapter must be pinned); numeric order puts every new chapter at the same place | 6/10 (+1 rename) | hand-kept |
| same | `surfaces` (Direction B roster) | authored | 0/10 | hand-kept |
| `.abcd/development/release-gate/receipt.example.json` | `manifestHash` | derived: sha256 of manifest.json | 9/10 | hand-kept |
| `.abcd/development/brief/04-surfaces/README.md` | end of the chapter table (`\| NN \| /abcd:<verb> \|`) | authored, numbered rows | 8/10 (7 new rows, 1 rename) | hand-kept |
| same | the `agents and hosts` help-group row, one line listing verbs | authored list on one line | 5/10 | hand-kept |
| same | the "one file per verb" list, wrapped prose | authored list | 5/10 | hand-kept |
| `.abcd/development/release/surface.json` | one entry per command path | derived | 10/10, each at its own sorted position | generated (`cmd/abcd-gen-surface`, drift-tested) |
| `docs/reference/cli/commands.md` | one section per command | derived | 10/10, each at its own sorted position | generated (`cmd/abcd-gen-cli-ref`, drift-tested) |
| `internal/core/surface/sentences.go`, `internal/surface/cli/helpgroups.go` | one map entry per verb | authored, sorted by key | 10/10, 8/10, each at its own key | hand-kept |
| `internal/surface/cli/cli.go` | one `root.AddCommand` line per verb | authored | 10/10, at different lines | hand-kept |
| `internal/README.md` | — | authored | 0/10 | hand-kept |

Every row touched at the same line by every branch is a guaranteed conflict
between any two concurrent surface branches: the five release-gate rows are
all of that kind, and four of them are values a machine can derive. The
sorted, one-entry-per-key files (surface.json, commands.md, sentences.go,
helpgroups.go) change in every branch but at different places, and conflict
only when two branches add neighbouring verbs.

Replaying every in-branch merge of the PRs merged since 2026-09-24 with `git
merge-tree` (438 merges, 132 with a conflict) undercounts, because a branch
that rebased leaves no merge to replay: manifest.json and receipt.example.json
conflicted in 3 each, commands.md in 17 and cli.go in 10, the last two from
edits to the same command's prose rather than from additions. #1004's last
merge of main (71f004830) conflicted on receipt.example.json alone.
