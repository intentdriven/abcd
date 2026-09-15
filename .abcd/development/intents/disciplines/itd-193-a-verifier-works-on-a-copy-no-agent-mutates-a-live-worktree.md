---
id: itd-193
slug: a-verifier-works-on-a-copy-no-agent-mutates-a-live-worktree
spec_id: null
kind: discipline
kind_notes: "Cross-cutting gate over how any agent verifies: it constrains the method of every review, probe, benchmark and mutation run rather than delivering a capability with a user moment. No verb, and no automation at this rung -- the documented protocol is the MVP, per the script-first-mvp principle. Adjacent to the one-writer-per-file principle, which governs committed records across merge windows and puts per-worktree files out of scope; this record governs the working tree during a single window, which that principle does not reach."
suggested_kind: discipline
reclassification_history: []
builds_on: []
severity: minor
impact: additive
origin: researcher-authored
production_mode: dictated-and-formatted
---

# A verifier works on a copy: no agent mutates a live worktree to test it, and the tree is proved clean before any act that reads it

## The rule

Verification that changes the thing being verified is not verification. Two
obligations follow, and they sit on different parties.

**On the verifier.** An agent that mutates code to see whether a test catches
the mutation — or that patches, stubs, or instruments a tree to probe it —
does that on a **copy**, never on the live worktree:

```text
git -C <worktree> archive HEAD | tar -x -C <scratch>/mut
```

The copy lives in the local ephemeral tier. Before reporting, the verifier
proves the original is untouched, and says so:

```text
git -C <worktree> status --porcelain    # must be empty
```

**On whoever acts on the tree.** A merge, a commit, a push, or a gate run is
preceded by that same emptiness check, taken **immediately before the act** and
not inherited from earlier in the sequence. A tree that was clean five minutes
ago is not evidence about a tree now.

## Why

The hazard is the window, not the intent. An in-place mutation is usually
restored within seconds, and the agent doing it is behaving carefully by its
own lights. But while it is applied, every other reader of that tree is wrong:
a gate reports on code nobody wrote, a merge takes a mutation into a branch, a
commit makes it permanent. Nothing announces the window, and the restore step
is itself fallible — a sibling agent lost a job to an interactive `cp -i`
prompt while restoring a file this way, leaving the mutation in place until it
was noticed.

It also breaks the peer-work convention from the other end. A concurrent
session that sees the modification cannot tell a mutation from real work, so
the correct response — leave it alone and report it — costs a round trip and
invites the wrong response, which is to revert a peer's genuine change.

## The gate

- **Given** an agent verifying by mutation, probe or instrumentation, **when**
  it applies the change, **then** the change lands in a scratch copy and the
  reviewed worktree is untouched.
- **Given** a verifier's report, **when** it is delivered, **then** it states
  that the reviewed worktree is clean, and names any file it could not restore.
- **Given** a merge, commit, push or gate run, **when** it begins, **then**
  the working tree is proved clean immediately before it, not earlier in the
  sequence.

## Fit

The discipline family is the right home: this fixes a method across every
verification, and has no user moment of its own. It is adjacent to the
[one-writer-per-file](../../principles/one-writer-per-file.md) principle and
does not overlap it — that principle governs committed records across merge
windows and puts per-worktree files explicitly out of scope, while this governs
the working tree inside a single window.

## Staging

**Reach is this repository only, and that is a real limit rather than a
rounding error.** A rule reaches every abcd-managed repo when it is a recall
keyword and rule set in a bundled default domain, and those are compiled into
the binary. This record is durable and citable here; it is not yet injected
anywhere else.

The rungs, smallest first, per script-first-mvp:

1. **The documented protocol** — this record, plus the line in the repository's
   own conventions. Live at filing. It is what agent briefs cite.
2. **A recall-injected domain rule**, so a session working on review or
   verification receives it without being told. In this repository that is one
   entry in `.abcd/rules.json`; reaching other managed repos means a bundled
   default domain, which is a code change and its own intent.
3. **A mechanical check.** The verifier half is observable — a clean-tree
   assertion at the end of a review — and the actor half is a precondition on
   the merge path. Neither is built, and neither should be built before the
   protocol has been run enough times to say what it costs.

Nothing above rung 1 is claimed. A brief that cites this record is the whole
of the enforcement today.
