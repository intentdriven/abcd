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
| `.abcd/development/brief/04-surfaces/README.md` | end of the chapter table, one row per chapter number | authored, numbered rows | 8/10 (7 new rows, 1 rename) | hand-kept |
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

## Rulings

Recorded here rather than in DECISIONS.md, by the agent fixing the record.

- **The example receipt's `manifestHash` becomes a placeholder,
  `sha256:<computed: the sha256 of manifest.json at subject.digest.gitCommit>`,
  rather than being dropped.** Nothing reads the example's literal as an
  input: `receipt_gate` reads only `.abcd/work/reviews/<sha>/`, `abcd docs
  fidelity record` computes the hash itself (`lint.ReleaseGateManifestHash`),
  the release workflow and launch's receipt tests build their own fixtures,
  and the only reader was `TestReceiptExampleManifestHashIsTheCommittedManifests`,
  which existed to keep the copy fresh. Keeping the field keeps the example a
  complete picture of the shape, which the README and `commands/launch.md`
  point readers to; a placeholder that is not hex can never pass for a hash,
  and the replacement test holds the example to it and proves `receipt_gate`
  refuses it, with the computed hash in its place as the control. The real
  gate is untouched: a release receipt whose `manifestHash` differs from the
  shipped manifest is refused exactly as before.
- **`checkerCount` is computed, not stored.** Nothing consumed it (the
  detector never read it), it drifted once already (iss-2609011423385217), and
  the merge test shows what a stored copy does under a merge: both branches
  bump 42 to 43, git takes the identical change as clean, and the merged
  manifest says 43 where the lists give 44. The test now refuses the field and
  states the count it would have held.
- **`promptHash` stays stored and verified; the rosters leave the prompt
  text.** Receipts echo it as `policy.promptHash`, so it has a consumer and
  dropping it would leave receipt writers computing it by hand. The command
  and agent rosters move out of the one-line `context` string into top-level
  `commands` and `agents` arrays, and the context names them through
  `${commands}` and `${agents}`, which the detector fills, as it already fills
  `${doc}` and `${s.name}`. `promptHash` keeps its definition word for word
  (sha256 over context, directionA, directionB) and now changes only when the
  wording does; the rosters are pinned by `manifestHash` with the rest of the
  file. The rendered context is the old one, names and order unchanged,
  wrapping aside (checked with node against the detector's own fill lines).
- **Lists are sorted, one entry per line, the order held by the test.** The
  rosters sort by name. `briefDocs` sorts by directory and then by file name
  with its leading chapter number removed: chapter numbers are handed out in
  sequence, so numeric order puts every new chapter on the line after the
  newest one, and two concurrent branches take the same number anyway. Slug
  order puts a chapter beside its neighbours by name, the same place its
  command takes in the roster. Two names that sort into the same gap still
  conflict, and the resolution is to keep both lines.
- **No `merge=union`.** It is unsafe on JSON: it keeps both sides' lines,
  so two branches appending to one array each add the comma to the old last
  entry and leave their own entry bare, and the union is
  `"x",` `"a"` `"b"`, invalid. A changed derived value would come out as two
  copies of the key. And the forge's
  merge queue ignores merge drivers anyway (iss-2609100507439414's
  corroboration of 2026-09-19).
- **Left alone:** `surface.json` and `docs/reference/cli/commands.md` are
  generated from the command tree, sorted by command path, and drift-tested,
  so additions land apart; their conflicts are edits to one command's prose,
  resolved by regenerating, which commands.md's header already says.
  `internal/README.md` was touched by none of the ten branches. The brief's
  `04-surfaces/README.md` register is the remaining shared-line hotspot (the
  numbered chapter table, the one-line `agents and hosts` row and the "one
  file per verb" list); it is authored prose in the design record, ordered by
  chapter number, and reordering it is left to its own change.
