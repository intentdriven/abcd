---
id: adr-2610222026262980
slug: coordination-between-headless-agents-is
status: proposed
date: 2026-10-22
supersedes: null
superseded_by: null
related_intents: [itd-2610170634394993, itd-2609091034175565, itd-2609151838312703]
related_rfcs: []
related_adrs: [adr-2610222026265719]
---

# ADR-2610222026262980: Coordination between headless agents is bounded by deadlines, fenced by the claim's tip, version-checked, and done under each session's own credential

## Context

The coordination intent (itd-2610170634394993) replaces the messages
sessions sent each other on 2026-10-16 with records on the shared remote that
any session reads and acts on. Its decomposition, confirmed by the product
thinker on 2026-10-17 (the calibration note under that date), routed four
rules to an ADR rather than to the intent, because each one is a boundary
other work must not cross, not a feature:

- That evening the load gate deadlocked between two builds of its own queue
  rule (iss-2610161634043273) and no wait timed out for eight hours.
- A claim taken over from a dead holder can be acted on by that holder if it
  comes back, unless a later act re-checks that the claim is still its own
  (Chubby's sequencers, Kubernetes leases; the research note
  `2026-10-17-parallel-headless-agent-coordination.md`).
- Two sessions on different abcd versions read each other's records; an
  older reader that guesses at a newer record can wait on it forever.
- Every session on the machine used the person's own forge login, which
  holds admin rights, so any session could change a setting meant for the
  person alone (the reservation cap, captured on 2026-10-17). The product
  thinker ruled the same day that autonomous sessions never work under that
  login, and on 2026-10-22 that unattended runs happen only where the login
  is absent (C1) and that each session's own App token ships with the
  coordination (C3).

The planning prep of 2026-10-18 settled the mechanics each rule rests on
(dcn-2610180720366282,
dcn-2610180720397970, dcn-2610180720379765, dcn-2610180723332655): the generation number is the claim
branch's tip, every expiry is derived when the records are read, and the
credential is a GitHub App token minted for the person.

## Decision

Four rules bind every coordination act abcd makes, by any verb, in any
session:

1. **Every wait has a deadline.** A wait on a slot, a dependency, a claim, a
   merge or another session ends by a stated time, and a wait that reaches it
   ends in a refusal that names what it waited on and for how long. No wait
   is open-ended, and no deadline depends on a process staying alive: each is
   derived from the records when they are read.
2. **A claim is fenced by its tip.** Every push, queue entry or tag made
   under a claim re-checks that the claim's branch tip is still the one the
   act started from. On a push the forge enforces it (a push that is not a
   fast-forward is refused); for a queue entry or a tag abcd re-reads the tip
   and refuses the act if it moved. A claim whose holder stopped renewing is
   retaken only after its expiry plus a fixed delay.
3. **A reader refuses a coordination record newer than it knows.** From the
   release that adds a version to the coordination records, a reader meeting
   a version above its own refuses the record with a message naming both
   versions, and never waits on it or guesses its meaning. Readers released
   before that ignore the field, which is why the rule starts there.
4. **An autonomous session never acts under the person's own forge login.**
   It pushes, opens pull requests and reads the forge with a short-lived
   token abcd's GitHub App mints for that session, limited to the repository
   and the acts its work needs. An unattended run starts only where the
   person's saved login cannot be reached (a cloud session, a container or a
   separate user account); abcd checks the known places a login is kept
   before starting one, and refuses to start, naming where it found one.

## Alternatives Considered

- **These four as one trust rule (chosen).** They are the four ways
  coordination by records fails silently (a wait that never ends, a stale
  holder acting, a reader guessing, a session holding more than it needs),
  and each is checkable at the act.
- **A coordinator process holding locks (etcd, a Temporal server).**
  Rejected in the state-of-the-art pass: out of scale for one repository,
  and a process that must stay alive is the failure 2026-10-16 showed.
- **Leave deadlines and fencing to each verb.** Rejected: the load gate's
  eight-hour deadlock was exactly a verb that had none.
- **Say plainly that the person's login is still within reach, rather than
  run where it is absent.** Offered to the product thinker on 2026-10-22 and
  not chosen: a caveat is not a guarantee.

## Consequences

- Every coordination verb (land, build, drain, status, reserve, the load
  gate) carries a stated deadline and a named refusal at its end; a test per
  verb watches it end.
- Queueing and tagging gain a tip re-check; a session whose claim was taken
  over cannot land under it.
- The coordination records gain a version field in the release that ships
  this; the refusal binds from then on.
- Unattended runs need a setup the person does once: the App registered in
  the browser (C3), and a place to run without their login. abcd states that
  as a step only the person can take.
- Stated in the brief's invariants once this ADR is accepted, which happens
  with the product thinker's sign-off of the intent's plan.
