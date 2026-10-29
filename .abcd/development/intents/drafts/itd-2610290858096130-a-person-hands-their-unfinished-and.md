---
id: itd-2610290858096130
slug: a-person-hands-their-unfinished-and
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609091718595846]
related_issues: [iss-2610290730196083, iss-2610290741142392]
severity: major
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# A person hands their unfinished and private work on a repository to another account, computer or colleague, and it arrives whole and private

## Press Release

> **A person moving to another computer or account, or passing work to a colleague, packs everything unfinished in one step, and it arrives whole, private and ready to carry on.** abcd packs the person's private labs, their private ideate verdicts and every piece of unfinished work: branches not yet on the shared remote, changes not yet committed, files not yet tracked, and the notes each worktree keeps locally. It writes one package that only the named receiver can open, with a generated note that says, branch by branch, how far each piece got and what comes next. On the receiving side abcd checks who sent it and that it belongs to the same repository, restores the work into that account's own private stores and worktrees, and overwrites nothing. If anything would clash, it writes nothing and lists every clash. Files that look like passwords or keys are left behind, and the package names each one. The work-in-progress holds the sender had on records pass to the receiver directly, so nobody else can take them in between. When the receiver is the same person, the sender's copy is cleared once the receiver confirms the work arrived, so the work lives in one place.
>
> "My laptop was leaving that morning with eleven unfinished branches on it," said Alice, a solo founder. "One step packed them. On the desktop, one step put them back, each in its own worktree, with a note saying where each stood. The one branch with an API key in a settings file came across without that file, and the note told me which file to bring over myself."
>
> "A colleague handed me a half-finished lab and two branches before going on leave," said Carol, a tech lead. "The package opened only on my account. Their holds on the records came to me with it, so the lane was mine the moment it landed."

## Why This Matters

A person's unfinished work on a repository lives in places only their own account can read: the private lab store, the private ideate store, local branches, worktrees with uncommitted changes, and each worktree's local notes. Moving it today is done by hand: a git bundle of the branches, a patch and an archive per dirty worktree, a hand-written status table and a long handover note. That route skips redaction and integrity checks, drops the owner-only modes the private stores rely on, and says nothing about what cannot travel: the receipts that prove a commit passed its gates, the state of a running build loop, and the holds on records. On the receiving side nothing takes the material in, so later lab, ideate or build runs there cannot read it.

The lifeboat already packs a repository's record and unpacks it elsewhere, but it is public by design: anyone may read it, and it carries the project's theory, not one person's resumable work. A handover is the private, resumable counterpart, addressed to one receiver.

## Mechanism

We expect a handover to make moving unfinished work a two-step act with nothing lost, because every piece that travels today by hand has a mechanical form. Branches and uncommitted work travel as commits in one git bundle. Private non-git material travels in the same envelope. The note is generated from the repository's own state, never typed. The package is sealed to the receiver and signed by the sender. A handover that leaves a piece behind without naming it, or that needs a hand-written note to be usable, shows the claim wrong.

## Scope Conditions

- The sender and receiver work on the same repository: the receiving checkout has the same root commit as the sender's.
- The receiver is the same person on another computer or account, or a colleague whose public key the sender can name.
- Work that cannot travel is named, not carried: preflight receipts (listed for information only), running build-loop state, and anything regenerable such as a lab's repository snapshot or built binaries.
- The package never passes through the repository or the shared remote.

## Acceptance Criteria

- **Given** a checkout with branches not on the remote, a worktree with uncommitted and untracked changes, private labs and private ideate verdicts, **when** the person packs a handover to a receiver, **then** one package is written that holds all of them and a generated note listing every branch with its base, its distance from the default branch, whether it was dirty, its open pull request, its holds and its next step.
- **Given** a package, **when** anyone other than the named receiver tries to open it, **then** it cannot be opened, and nothing in it is readable.
- **Given** a package whose signature does not match the sender's published key, or whose repository root commit differs from the receiving checkout's, **when** the receiver runs the receive step, **then** it refuses, names which check failed, and writes nothing.
- **Given** a receiving checkout where a branch of the same name exists at a different commit, **when** the receive step runs, **then** it writes nothing at all and lists every clash in one report; a branch already at the same commit is skipped, so a second receive of the same package changes nothing.
- **Given** an unfinished branch holding a file that looks like a secret, **when** the handover is packed, **then** that file is left out, the pack goes ahead, and the note names the file and why it stayed behind.
- **Given** agent session transcripts for the work, **when** the handover is packed, **then** they travel redacted, with no absolute local paths and no unredacted personal names.
- **Given** a sender holding a record that a colleague receives, **when** the receiver accepts the package, **then** the hold passes to the receiver in one act, with no moment at which a third party could take it.
- **Given** a handover to the same person, **when** the receiver confirms arrival, **then** the sender's copies of the branches, worktrees and private stores are cleared and the clearing is logged; before confirmation nothing is cleared.
- **Given** private material arriving, **when** it is restored, **then** it lands only under the receiver's own home store with owner-only permissions, never inside a working tree.

## Decisions

The product thinker decided on 2026-10-29:

- A colleague can receive a handover in the first version, not only the person themselves, so a transfer of holds on records is part of it.
- Files that look like secrets are left behind and named by the note. They don't stop the pack.
- Agent session transcripts travel, redacted.
- A handover to oneself is a move, not a copy: the sender's copy is cleared once the receiver confirms arrival.
- The new Go dependency the age format needs (the pure-Go `filippo.io/age` library, BSD licence) is signed off, as AGENTS.md requires before any new dependency is added.

Technical choices with one defensible state-of-the-art answer, recorded rather than asked:

- Uncommitted and untracked work is captured as a commit on a reserved ref inside one git bundle, not as a patch plus an archive. A bundle carries refs and objects but no index or working tree ([git-bundle](https://git-scm.com/docs/git-bundle)), and git itself now moves stashes as commits ([git-stash export](https://git-scm.com/docs/git-stash)).
- The package is encrypted to the receiver with the age format ([C2SP age](https://github.com/C2SP/C2SP/blob/main/age.md)), which accepts SSH keys and a forge user's published keys as recipients ([age](https://github.com/FiloSottile/age)). The sender's identity is proven by an SSH signature over a manifest of every file's hash, under a namespace of its own ([ssh-keygen -Y](https://man.openbsd.org/ssh-keygen#Y)), because age itself does not authenticate the sender.
- Receiving lands refs in a holding namespace first, then adopts them where nothing exists, under the lifeboat embark's rule: refuse the whole write on any clash, and skip identical bytes.
- Holds pass by an explicit transfer act. Releasing and re-reserving leaves a window for a third party, and waiting for a lease to lapse costs up to a day ([Kleppmann on leases](https://martin.kleppmann.com/2016/02/08/how-to-do-distributed-locking.html)).
- The note is generated as data with a readable rendering, never hand-written, and receiving ends with an acknowledgement, as shift handoff practice requires ([Google SRE book](https://sre.google/sre-book/managing-incidents/)).
- The handover is a sibling of the lifeboat, not an extension of it. They share the destination-safety, secret-scan, hashed-manifest and contained-write gates, but the lifeboat stays public and the handover stays private.

## Open Questions

- The shipped one-file session artefact this builds on carries two known defects from its audit: absolute paths inside quoted transcript text, and metadata that can forge the document's structure. Both must be fixed, or the artefact scanned, before transcripts travel.
- Related drafts: the lifeboat that packs from a worktree (itd-2609090746410233) keeps the abandoned-work case and builds on this handover's worktree packer. The lifeboat that packs from a lab home (itd-2609090746414083) shares its lab packer. Two sessions sharing one run (itd-2609221656373558) stops at another account or computer, which is the question this intent answers.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
