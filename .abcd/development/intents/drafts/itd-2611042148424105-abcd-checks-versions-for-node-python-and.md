---
id: itd-2611042148424105
slug: abcd-checks-versions-for-node-python-and
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2611041815429986]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
blocked_by: [itd-2611041815429986]
---

# abcd checks other languages' tools and names a missing one

## Press Release

> abcd checks versions for Node, Python and Rust tools as it does for Go, lets two projects use different copies of one tool, and names a tool a stage failed to find. Version files stay plain data, and any that names a program to run is refused. "Our project pins Node in one file and Python in another," said Carol, a platform engineer. "abcd told me before the run that my Node was a major version behind, and it never ran anything from those files to find out."

## Why This Matters

itd-2611041815429986 checks any listed tool's file and fingerprint and checks a version
for Go alone. A repository that pins Node, Python or Rust gets no version check, and two
repositories that need different copies of one tool cannot both be served. A tool nobody
listed still fails mid-run with a bare `command not found` buried in a log.

Version files are where the risk lies, since some name a binary to run rather than a
version. [The research note](../../research/notes/2026-11-04-repository-tools-on-path-sota.md)
lists the forms, and adr-2611041815445203 is the rule.

## Mechanism

We expect fewer wrong-version failures, because each language's own files say what is
needed and a strict grammar reads them without running anything. This would be shown
wrong if real projects' version forms mostly came back unrecognised, or if the upkeep of
the grammars outran the failures they prevent.

## Scope Conditions

- Holds for the forms each grammar recognises. Anything else is reported as unrecognised and is not enforced.
- Holds for a real program. A version manager's launcher is never run, so its version stays not verified, which is common for nvm and shim users.
- Holds with each tool's own closed set of version files, under the rule of itd-2611041815429986.
- Holds for Node, Python, Rust and `.tool-versions` at first, with no standard precedence among Node's files, so the report names the file it read.
- Holds for a stage's exit code as evidence of a missing tool, not for spotting an unlisted tool in advance.

## Acceptance Criteria

- **Given** a version file a listed tool points at names a binary or a remote, such as `path:` or `ref:` in `.tool-versions`, a rustup path toolchain or a URL in `packageManager`, **when** abcd reads it, **then** the start is refused and nothing it names is run.
- **Given** a version in a form abcd does not recognise, such as `lts/*`, **when** abcd reads it, **then** the report calls it unrecognised and it is never used to compose a path.
- **Given** a project lists `node`, reserved by itd-2611041815429986 because a harness may run on it, **when** a runner launches an agent, **then** the harness starts on the copy it would have used, and the project's commands get the registered copy.
- **Given** a version file names a tool not on the list, such as a `.nvmrc` used only by a docs site, **when** the test runs, **then** the run proceeds, and the report warns and proposes adding it.
- **Given** a probe of a Node, Python or Rust copy, **when** abcd asks its version, **then** each tool's no-download switch is set and nothing is fetched, whatever the repository asks for.
- **Given** two repositories on one computer need different copies of one tool, **when** a person registers a second copy for one of them, **then** that repository uses it and the computer's copy serves the rest; the person says where in both cases.
- **Given** a stage exits 127 and its log names a command, **when** the run reports it, **then** it is named as a confirmed missing tool and nothing is refused.

## Decisions

Recorded with no question put. Technical facilitator on delegation, 2026-11-04:

1. The scan of changed scripts first drafted, looking for commands nobody listed, is dropped. The security review found the shell tokenizer is not built for it and that it would be wrong in both directions. Exit 127 is confirmed evidence, so it stays.

## Open Questions

- Which language comes after Go, from what real repositories need first (technical facilitator).

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
