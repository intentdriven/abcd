---
id: itd-2611041815433220
slug: a-person-whose-repository-needs-a-tool
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

# abcd installs a repository's missing tool on a per-tool yes

## Press Release

> A person whose repository needs a tool that is nowhere on the machine can have abcd fetch it after saying yes to that one tool and that one version. Grace, a CTO, opens a project that needs a Go version they have not installed, and abcd explains what it would fetch, from where, and how it was checked. "It asked about that one thing, I said yes, and I did not have to look anything up," said Grace. A repository can ask for a version and can never trigger the download.

## Why This Matters

The check of itd-2611041815429986 ends with a refusal when no suitable copy is on the machine,
which leaves the person to find and install the tool. abcd's explain-then-install
mode (itd-63) already asks a per-tool yes for tools abcd itself uses.

It cannot serve this as it stands. adr-2610241057124151 fetches only fixed data
compiled into the binary, never composed from a repository file, while a
repository's tool is whatever version its files name. Filed apart from the check
on the product thinker's choice of 2026-11-04 to take finding before installing.

## Mechanism

None stated.

## Scope Conditions

- Holds only after itd-2611041815429986, whose list says what is needed and whose test says what is missing.
- Holds only on a person's yes for one tool and one version, never unattended and never in CI.
- Holds only where the download can be checked; where the check comes from the same origin as the file, the question says so.

## Acceptance Criteria

- **Given** a repository needs a version nowhere on the machine, **when** the person is offered the install, **then** the question names the tool, the version, where it comes from, how it is checked and where it will live, and nothing is fetched without their yes.
- **Given** a yes for one version, **when** the repository later asks for another, **then** abcd asks again.
- **Given** an unattended run, **when** a tool is missing, **then** no download happens and the run is refused.
- **Given** a yes at a terminal, **when** the install finishes, **then** the copy is registered with its fingerprint, because the question named where it would live and the yes accepted that, and a yes off a terminal is refused, as registering is.

## Open Questions

- Whether to use each tool's own mechanism where it is checked (Go's checksum database needs a `go` already present), print the vendor's command, or widen adr-2610241057124151 (technical facilitator; a change to that record is the product thinker's).
- Which tools come first, and what is said about rustup, which does not validate signatures (technical facilitator).

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
