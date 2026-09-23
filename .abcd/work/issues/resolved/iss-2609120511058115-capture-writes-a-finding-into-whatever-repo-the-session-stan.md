---
schema_version: 1
id: "iss-2609120511058115"
slug: "capture-writes-a-finding-into-whatever-repo-the-session-stan"
severity: "major"
category: "ux"
source: "agent-finding"
found_during: "peer report of nine misfiled installer records, 2026-09-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/validate.go"
resolution: "capture refuses a --found-at that names a repo-relative path not present in the checkout (or leaving it), before the ledger is touched; conceptual locations and an absent value are written as given"
impact: fix
resolved_by:
  commit: "6479a330"
---

Nine findings about abcd's own installer are sitting in a teaching-materials
repository's ledger. Reported by a session working in that repository, which
could see the records but not the cause.

## The cause, established

**Not a containment bug.** The issue ledger is directories under the repository's
own ledger root (`internal/core/issueschema/ledgerdirs.go`), with no
root-commit-SHA keying — that keying belongs to the user-level stores
(transcripts, worktrees, voyage) and never to the in-repo ledger. So
`gitutil.CheckoutRoot` resolved correctly and wrote to the repository it was
standing in. The capturing session simply had that directory as its working
directory.

The store worked. The distinction matters because the two hypotheses have
opposite remedies, and the reporting session could not tell them apart from where
it stood: a wrong store resolution would be a containment defect in the resolver,
and this is not that.

## What is worth fixing anyway

`abcd capture` will write a finding about anything into whatever repository it is
standing in, and **nothing it records has to be true of that repository**.

`found_at` is the field that could have refused these nine. It is optional and
unvalidated: `internal/core/capture/validate.go:103` lists it among
`found_at, lapsed_at, details, suggested_fix, wontfix_reason, resolution,
promoted_to` as optional strings, and nothing checks that the path it names
exists in the tree. All nine carry it empty.

So the failure is silent at both ends. Nothing refuses the write, and afterwards
nothing distinguishes a record about this repository from a record that merely
landed here: the reporting repository's board now reads "open 85" and leads its
recent-open list with an abcd installer defect as that repository's top open
issue.

## The shape of a guard

A path is checkable, and semantics are not. Do not try to judge whether a finding
is "about" this repository — that is not mechanical and a gate that guesses it
would be worse than none.

What IS mechanical: **when `--found-at` names a repo-relative path, require it to
resolve in the tree.** That alone would have refused all nine of the misfiled
records had they named the installer files they are about, and it costs nothing
in the ordinary case where the path is real. It also makes the empty value the
only way through, which is the right place to put a nudge rather than a refusal —
a capture with no `found_at` is legitimate (a conceptual finding, a process
observation) and must stay legitimate.

The weaker companion: the conceptual-location escape the field already documents
("optional repo-relative path or conceptual location") means a non-path value
must stay accepted, so the check fires only on something that looks like a path
and does not resolve. Getting that discrimination right is the work; refusing
every non-resolving string would break the documented use.

## Related

- `iss-2609120505141653` — the same nine records assert `origin:
  researcher-authored`, which claims a person invoked the verb.
- `iss-2609120452071388` — `--source` and `--category` do not name their members
  in help, and all nine carry the catch-all `observation`.

Three findings from one batch of misfiled records, each a different surface
failing to hold a caller to something it knows.

## Acceptance

- **Given** a capture whose `--found-at` names a repo-relative path that does not
  resolve in the tree, **when** the verb runs, **then** it is refused and nothing
  is written.
- **Given** a capture whose `--found-at` is absent, or names a conceptual
  location rather than a path, **when** the verb runs, **then** it is written as
  it is today.

## Grounds

- pursued: a capture whose found_at names a path absent from the checkout is refused with nothing written, while conceptual locations still file; TestCaptureFoundAtMustResolveInTheTree holds both sides, and a real path refused or a conceptual phrase refused would show it wrong
