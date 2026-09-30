# Upgrade to v0.12.0

v0.12.0 is a breaking release. It changes two things a script, a hook or a
host driving abcd may depend on: the implement loop names a lane's stages
`stage` rather than `step`, and the command spellings that answered with their
replacement for one release are gone. This guide lists every change and what to
use instead.

## Update the loop's JSON readers

A spec lists its **steps**; the build loop lands each step as one lane and takes
that lane through its **stages**: `worktree`, `brief`, `implement`, `validate`,
`land`. `abcd implement step` keeps its name and performs one stage. Every field
that carries a lane's stage says `stage`:

| Where | Field in v0.11 | Field in v0.12.0 |
|-------|----------------|------------------|
| `abcd implement step --json`, `abcd implement receipt --json` | `performed` | `performed_stage` |
| `abcd implement step --json`, `abcd implement receipt --json` | `step` | `stage` |
| a loop refusal under `--json` (`abcd build`, `abcd implement step`, `receipt`, `status`) | `refusal.step` | `refusal.stage` |
| the run state file, `.abcd/.work.local/run/<run-id>/state.json`: each lane | `step` | `stage` |
| the run state file: each line of `record` | `step` | `stage` |
| the status board, `abcd --json`: a lane under `status.now` | `lane.step` | `lane.stage` |
| `abcd implement check --json`: the verdict | `step` | `stage` |
| the run log (`~/.abcd/runs/<root-sha>/<date>.jsonl`): a `refusal` line from `implement check` | `step` | `stage` |

The spec's own fields keep their names: `spec_step`, `step_title` and
`pending`. The text forms say "stage" for the lane's, as in
`completed lane-1's worktree stage` and `next stage: brief`.

`abcd implement check` calls what a session asks about a stage too. Its operands
keep their spellings (`lane`, `release`, `review`, `audit`, `land`), its text
says `<session> may take the review stage`, and an operand outside them is
refused as an `unknown stage`.

A run started with an older abcd carries on. Its state file (schema version 1,
the version v0.11.1 writes) is read as it stands, each `step` taken as the
lane's `stage`, and the read writes nothing; the run's next `abcd implement
step` or `receipt` writes the file at schema version 8. An older abcd cannot
read a version-8 file, so finish a run with the abcd that is going to keep
driving it.

## Replace the removed command spellings

Each of these is an unknown command in v0.12.0, refused with exit 2:

| Spelling | Use instead |
|----------|-------------|
| `abcd ahoy dry-run` | `abcd ahoy --dry-run` |
| `abcd ahoy identity-check` | `abcd ahoy --identity` |
| `abcd version` | `abcd --version` |
| `abcd version --check` | `abcd update --check` |
| `abcd docs lint` | `abcd lint docs` |
| `abcd site check` | `abcd lint site` |

Two parents keep their sub-verbs, and bare they list them and exit 0 instead of
naming a replacement:

| Bare spelling | For the report it gave, use |
|---------------|-----------------------------|
| `abcd identity` | `abcd lint identity` (`abcd identity render` and `abcd identity init` are unchanged) |
| `abcd ahoy remote` | `abcd ahoy --remote` (`abcd ahoy remote apply` is unchanged) |

Search your scripts, hooks and CI workflows for each spelling in the left-hand
columns, change it to the right-hand one, and run the changed command once to
see it answer.
