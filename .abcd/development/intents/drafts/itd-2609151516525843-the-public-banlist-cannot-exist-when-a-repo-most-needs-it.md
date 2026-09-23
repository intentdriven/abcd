---
id: itd-2609151516525843
slug: the-public-banlist-cannot-exist-when-a-repo-most-needs-it
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-74]
severity: major
impact: additive
promoted_from: iss-2609100506269348
origin: extracted-from-record
production_mode: hand-written
related_adrs: [adr-56]
---

# A Repository Can Ban A Name On Its First Commit, And A Machine Can Ban One Everywhere

Typed links: `promoted_from` [iss-2609100506269348](../../../work/issues/open/iss-2609100506269348-the-public-banlist-cannot-exist-when-a-repo-most-needs-it.md) (the finding: on a fresh public repository the committed banned-names layer cannot be created, and that window is exactly when a repository is being set up to ban a name); `builds_on` [itd-74](../shipped/itd-74-name-banlist.md) (the shipped two-layer banlist this extends — the committed CI-enforced layer and the gitignored per-machine one, described in [`04-surfaces/20-banlist.md`](../../brief/04-surfaces/20-banlist.md) and [`commands/banlist.md`](../../../../commands/banlist.md)); `refines` [adr-56](../../decisions/adrs/0056-an-exclusion-control-asserts-only-what-it-can-prove.md) (an exclusion control asserts only what it can prove — today the public layer's answer on such a repository is to report NOT ENFORCEABLE and write nothing, which is the honest answer and the whole of the remedy; this intent gives the control something it *can* prove, so the honest answer stops being "no layer at all"). Prose cross-references, not typed links, because no schema field carries the relation ([iss-2609091256264547](../../../work/issues/open/iss-2609091256264547-three-of-the-four-mandated-typed-relations-cannot-be-written.md)): [iss-223](../../../work/issues/open/iss-223-public-visibility-fence-vs-committed-record-repos.md) and its unfilled promotion [itd-159](itd-159-public-visibility-fence-vs-committed-record-repos.md), which state the committed-record declaration this intent's first half implements — the fence hiding records that are *already* committed is the other end of the same mechanism; [iss-176](../../../work/issues/resolved/iss-176-public-banlist-family-unenforceable-under-public-visibility.md), the resolved record that ends with the three candidate reconciliations a human was left to pick between; and [itd-150](../shipped/itd-150-agent-worktrees-commit-without-the-private-name-guard-abcd-w.md), whose worktree gap is a second symptom of the private layer being scoped to a checkout.

## Press Release

> **A repository can declare, in a committed file, that its record tier is meant
> to be committed — and the banned-names layer it needs on day one becomes
> writable on day one. And the names a person can never publish are declared once
> for the whole machine, not once per repository.** The visibility fence narrows
> on a declaration the repository can make before it has committed anything,
> instead of waiting on tracked files the fence itself prevents. Without that
> declaration the fence refuses exactly as it does today, and the refusal names
> the declaration as the way through rather than leaving the escape to be
> guessed. Alongside it, a private banned-names list lives in the user-level home
> and applies to every repository on the machine: a hostname, a device name, a
> client's project name is entered once and guarded in every checkout, including
> ones that do not exist yet. The two layers keep the boundary that makes them
> work — the committed one is what CI reads, the machine-global one is never read
> by CI, never committed, and never printed.
>
> "I set up repositories for other people all week, and the first thing each of
> them needs is a name it must never publish — which is precisely the moment the
> tool told me it could not write the list," said Jack, a consultant. "Now the
> repository says once that its record is committed and the list is writable
> immediately. And the handful of names that are mine rather than the client's —
> my machines, my network — I declared on this laptop, once, and every repository
> I touch is covered."

## Why This Matters

Graduated from [iss-2609100506269348](../../../work/issues/open/iss-2609100506269348-the-public-banlist-cannot-exist-when-a-repo-most-needs-it.md).
On a repository whose visibility is `public`, `ahoy install` writes a `.gitignore`
fence over the whole record namespace. `banlist add --public` then refuses,
correctly, because a config written under that fence would reach no CI run. The
fence narrows to the local tier only once the namespace already holds tracked
files, because narrowing needs positive evidence — and a brand-new repository has
none. The guarantee is therefore unavailable in exactly the window that creates
the need for it: a rename or an extraction, which is precisely a new repository
with nothing tracked yet and a name it must never publish.

The escape exists and is not discoverable: commit the record tiers first with
`git add -f`, against the tool's own fence, then re-run install so the fence
narrows, and only then write the store. Three steps, one of them forcing past a
gitignore the tool wrote a moment earlier, none of them named by the refusal. The
offered alternative — ban the name on the private layer instead — silently drops
CI enforcement, which is the entire point of the committed layer.

That is a bootstrap paradox rather than a bug, and every route through it runs
into what public visibility is declared to mean. Three candidates were left for a
human to pick between in [iss-176](../../../work/issues/resolved/iss-176-public-banlist-family-unenforceable-under-public-visibility.md):
move the config outside the namespace, carve a single un-ignore into a table that
has no exceptions, or declare the committed layer private-visibility-only.
[iss-223](../../../work/issues/open/iss-223-public-visibility-fence-vs-committed-record-repos.md)
states a fourth shape from the other end — a committed-record declaration that
suppresses the fence — and its promotion [itd-159](itd-159-public-visibility-fence-vs-committed-record-repos.md)
is still an unfilled draft. **The product thinker's ruling on 2026-09-15 takes
that fourth shape, and takes a second capability with it.**

**Half one: a committed declaration lifts the fence.** A repository says, in a
file the fence does not cover, that its record tier is meant to be committed.
That statement is evidence a repository can give on its first commit, which is
what detection of tracked files can never be. With it present, install narrows
the fence and the committed banned-names family is writable straight away.
Without it, nothing changes: the fence stands, the refusal stands, and the status
board still reads NOT ENFORCEABLE — this adds a door, it does not weaken a wall.

**Half two: a machine-global private list.** A person's most sensitive banned
names are not a property of a repository at all. Hostnames, device names, network
prefixes, a private project's name: they are the same in every checkout on the
machine, and today they are declared once per repository in
`.abcd/.work.local/private-names.txt` — which means a new checkout starts
unguarded, and the warning that says so is easy to read past. abcd already has a
user-level home for machine-scoped state, keyed where a repository is involved on
the repository's root commit. Nothing in it holds a banlist, and nothing today
applies a banned name across every repository on a machine. The product thinker
put it as a question — the home exists, why is the list not there, or split into
a global section and a per-repo one — and the answer is that nobody has built it.

**It was made explicit before the ruling that half two cannot rescue half one.**
A list in the user's home is invisible to CI by construction: a continuous
integration runner clones the repository and has neither the home nor any right
to it. So the machine-global layer cannot enforce anything for anyone but the
person whose machine it is, and it does nothing for the committed layer's reach.
The product thinker chose both halves with that limit stated: the declaration
makes the *committed* layer creatable when a repository most needs it, and the
machine-global list stops a person re-declaring their own names in every
repository they open. Neither substitutes for the other, and saying so is part of
what ships.

The honesty rule this extends is [adr-56](../../decisions/adrs/0056-an-exclusion-control-asserts-only-what-it-can-prove.md):
a control asserts only what it can prove. Today the public layer's compliance with
it is total and unhelpful — it can prove nothing about a fenced config, so it
writes none and says so. The refinement is to give the control a fact it can
examine. What must not move is the claim itself: a machine-global list must never
be described, on any surface, as protecting anything a CI run sees.

## Mechanism

We expect a committed declaration to close the bootstrap paradox **because the
fence narrows on evidence, and a declaration is the one kind of evidence a
repository can produce before it has committed anything** — where detection of
tracked files is, by construction, the kind it cannot. The claim is falsifiable in
the artefact rather than in a statistic: if the declaration's own home sits inside
what the fence covers, it cannot be committed either and the paradox has been
relocated rather than closed. So the declaration lives outside the fenced
namespace, or the fence exempts exactly it and nothing else, and an acceptance
criterion below is written to catch a declaration that cannot be committed on a
fresh repository.

We expect a machine-global private list to remove the per-repository cost of
declaring a name **because the names in question are properties of the person and
the machine rather than of any repository**, so one declaration covers every
checkout including the ones that do not exist yet. The falsifier is a boundary
crossing in either direction: if CI can read the home list, it is a published
pattern and the layer has become the committed layer with worse properties; if any
pattern from it reaches a committed file, the layer has leaked the exact class of
string it exists to contain. Both are acceptance criteria rather than notes.

## Scope Conditions

- **Repositories using git, and the forge CI model where a runner sees a clone
  and nothing else.** The claim that CI cannot read a home list rests on that
  model; a runner executing on the person's own machine with their home mounted
  is outside it, and would break the boundary rather than test it.
- **Single-user machines.** The machine-global list is scoped to one person's
  home, which is where a machine's private names belong when the machine has one
  user. A shared or multi-user home is explicitly **out of scope** — see Open
  Questions.
- **The private layer's existing reach limits carry over unchanged.** A hook sees
  only the commits git asks it about, so a fast-forward pull, a rebase, an `am`, a
  revert, a cherry-pick and `--no-verify` bypass it exactly as they do today. A
  machine-global list widens *which repositories* a declared name is checked in;
  it changes nothing about *which operations* run the check.
- **Public-visibility repositories, for the declaration half.** A private
  repository commits the whole namespace already and has no fence to lift.
- **Machines that have opted the guard in.** A committed hook is not an armed
  hook; arming stays the clone's own step.

## Acceptance Criteria

> _BDD (the [itd-1](../disciplines/itd-1-acceptance-gates.md) discipline)._

- **Given** a fresh public repository with nothing tracked under the record
  namespace and the committed declaration present, **when** the committed
  banned-names layer is created, **then** it is written, git reports its path as
  tracked rather than ignored, and the status board reports the layer as
  enforceable — with no `git add -f` and no second install run anywhere in the
  sequence.
- **Given** the same fresh public repository **without** the declaration,
  **when** the same write is attempted, **then** it is refused exactly as today,
  the status board still reads NOT ENFORCEABLE, and the refusal names the
  declaration as the supported way through — so the escape is stated by the
  refusal rather than discovered.
- **Given** the declaration present on a repository, **when** install runs,
  **then** the fence narrows to the local tier in the same pass that writes the
  store, and the receipt says the narrowing happened and why.
- **Given** a name declared in the machine-global list, **when** a commit
  matching it is attempted in any repository on that machine — including one
  cloned after the entry was made — **then** the guard refuses the commit and
  names the entry's key alone, with the matched string and the pattern reaching
  no output, log or process argument.
- **Given** all three layers hold an entry for one name, **when** the layers are
  rendered and when the guard runs, **then** the precedence between the committed
  layer, the repository-local private layer and the machine-global private layer
  is the one the documentation states, and a test asserts it — including the case
  where a repository-local entry and a machine-global entry disagree.
- **Given** a CI run over a clone of a repository whose author has a populated
  machine-global list, **when** the pipeline executes, **then** no committed file
  and no CI-visible configuration references the home list, the run's result is
  identical to a run on a machine with no such list, and every surface that
  describes the layer says in its own words that CI cannot read it.
- **Given** a populated machine-global list, **when** the repository is inspected
  after any abcd verb has run, **then** no pattern from that list appears in any
  committed file, in any file staged for commit, or in any rendered output — the
  render shows keys only, exactly as the repository-local private layer does.
- **Given** the machine-global list is absent or holds no usable entries,
  **when** the guard runs, **then** it says so as loudly as an absent
  repository-local store does — an absent layer never looks like a present one,
  and a machine-global layer that silently checks nothing is the worst outcome
  available.

## Open Questions

- **Where the declaration lives, and what it is.** A key in the repository's
  committed config, a marker file, or a value of the visibility switch itself
  (a third setting meaning "public, record committed"). The constraint the
  mechanism imposes is that it must be committable on a repository with nothing
  tracked under the fenced namespace; within that, the shape is open.
- **Whether the declaration is an exception to the one-switch invariant or a new
  value of the switch.** Invariant 5 in
  [`02-constraints/03-invariants.md`](../../brief/02-constraints/03-invariants.md)
  says visibility is one switch with no per-subdirectory exceptions, and
  [iss-176](../../../work/issues/resolved/iss-176-public-banlist-family-unenforceable-under-public-visibility.md)
  named a single un-ignore as the candidate that would cost exactly that property.
  Reading the declaration as a second value of the switch keeps the invariant
  intact; reading it as an un-ignore does not. This intent does not get to choose
  that on its own — it is a ruling, and it is owed before the spec.
- **A shared or multi-user home is out of scope, and the product thinker is
  thinking it through.** What a machine-global list means when the home is shared
  between people, or when a machine is a shared build host, is a separate question
  with its own trust boundary. Nothing here should be built in a way that quietly
  answers it.
- **One store with two sections, or two stores.** The product thinker raised both
  shapes — a machine-global store beside the per-repository one, or a single JSON
  document with a global section and a per-repository section. The second reads
  tidily and puts the machine's most sensitive strings into a file format a second
  writer already shares; the first keeps the blast radius of a mistake inside one
  file. The existing store's format declaration and its second writer (the sources
  corpus sync) are what a decision here has to survive.
- **Whether the machine-global layer subsumes the worktree gap.**
  [itd-150](../shipped/itd-150-agent-worktrees-commit-without-the-private-name-guard-abcd-w.md)
  exists because the per-repository private store is per-worktree, so an agent's
  linked worktree commits with the layer absent; a machine-global list is read
  from the home and would cover that case without the primary-checkout fallback.
  Whether that retires itd-150, narrows it, or leaves it untouched is a question
  for whoever plans this — the two records answer different halves and neither
  refines the other today.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
