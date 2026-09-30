# Prefer SOTA

**The rule.** The state of the art is the default bar for every design,
tooling, or practice decision: where a choice has a credible SOTA answer, that
answer is the presumptive pick. But "it is SOTA" is never on its own sufficient
warrant to adopt. Before adoption, SOTA is challenged for *fit* against this
repo's stated preferences by an adversarial reviewer whose sole job is to find
where the generic best practice collides with a deliberate local constraint.
Only the SOTA that survives that challenge — or is re-selected in light of it —
is adopted. When the best option is not obvious, the sequence is fixed and
ordered: adversarial fit-challenge first, then a SOTA verdict that already
accounts for the challenge, then the pick.

**Why.** abcd is an *opinionated* configuration layer for development — its
value is the set of considered defaults it holds and propagates, not raw
novelty. SOTA is the right starting bar because it encodes the field's
accumulated learning cheaply; refusing it reinvents. But adopting SOTA
unfiltered is the opposite failure: a generic best practice can directly
violate a load-bearing local choice (host-delegated LLM work, minimal
dependencies, host-agnostic prose, a single source of truth). Running the
fit-challenge *before* the verdict is what stops "it's SOTA" from laundering a
preference violation — the adversary supplies the repo's own objections as an
input to the recommendation, so the verdict is fit-aware rather than fit-blind.
This is the opinion in "opinionated configuration": abcd does not chase novelty,
and it does not ossify — it takes the field's best and earns each adoption
against what the repo already believes.

**Bounds.**

- Binds on genuine forks — a design, tooling, or practice decision where the
  best option is not obvious. Trivial or fully-specified choices with a
  conventional default state the assumption in one line and proceed; they do
  not summon the process.
- The adversary challenges *fit*, not SOTA's abstract correctness: its output
  is the enumerated places generic SOTA would trample a *named* local
  preference, plus the surviving-fit hypothesis. The SOTA verdict then ranks
  options already filtered through that challenge; it never reports raw SOTA as
  the answer.
- A local preference can itself be wrong. When the challenge exposes a
  constraint that survives only by inertia, the response is to revisit it
  deliberately — an ADR under [`../decisions`](../decisions) — never to
  silently keep it and never to silently break it.
- Composes with [evaluator-outside-the-loop](evaluator-outside-the-loop.md)
  (the challenger is independent of whoever proposes the option) and
  [verifier-selects-gates-decide](verifier-selects-gates-decide.md): the SOTA
  verdict is a proposal; the human's adoption is the gate.

**Remedies proposed for an issue or chosen in an autonomous run.** Every
remedy cites its grounds, whoever writes it and whether or not a human is in the
loop: the remedy a capture proposes for an issue, and the pick a run with no
human in the loop makes (a fix for a captured issue, or one option over another
when a finding admits several). The grounds sit in the record that carries the
remedy (the issue's remedy line, the resolve note's grounds, the commit
message). Where the fix depends on outside practice (an interface convention, a
security posture, a tool's documented behaviour), the grounds include a
primary-source state-of-the-art check: the standard, the vendor's own
documentation or the practitioner research, named so a reader can open it.
Where the fix is purely internal, the grounds say so. A remedy suggested or
picked without that check can repeat a known-bad approach, and a record that
carries no grounds leaves nobody able to tell why the remedy beat the
alternatives. The check is the short form of this principle, not the full
adversary pass: it binds every remedy proposed or chosen, while the
fit-challenge above binds the genuine forks.

**Scope.** This is a propagated default, not a repo-local habit — it holds for
every abcd-managed repo, as one of the opinionated conventions abcd installs.
Its home in a managed repo is the marked working-conventions section abcd writes
into that repo's `AGENTS.md`; abcd's own record carries the normative statement
here.

**Live instance.** Choosing how to wire the iss-35 semantic cross-check as a
release gate: an adversarial reviewer challenged generic release-engineering
SOTA (policy-as-code, release-automation tooling, an LLM judge in CI, an
executable gate target) against this repo's host-delegated-LLM,
minimal-dependency, single-source-of-truth, host-agnostic preferences, and only
the fit-surviving form was taken to a SOTA verdict for adoption.

**Promotion.** MVP: a documented decision protocol — this principle, plus the
standing model-routing note naming which agents fill the challenger and
researcher roles. Tool: an abcd verb that scaffolds the adversary → SOTA-verdict
pass for a decision and records the challenge and verdict as a dated research
note beside the ADR it informs, so the fit-filter is a cheap repeatable step
rather than a remembered discipline. The managed-repo propagation path (the
`AGENTS.md` conventions section abcd installs) is itself the tool rung for
reach: the principle ships as an abcd default rather than being re-derived per
repo.
