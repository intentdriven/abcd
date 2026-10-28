# Coordination lives on the shared remote, never only in conversation

**The rule.** A fact one agent relies on from another (who holds which
piece of work, which change waits on which, whose turn it is to merge, how
much of a shared budget is left) is written where every participant
already looks: a record on the shared remote, or the forge's own state.
A message between sessions may announce such a fact, but the fact is never
true only because a message said it, and nothing waits on a promise made
only in conversation. Whoever needs the fact reads the record, and any
session can act on what the records say, so no one session has to stay
alive for the rest to go on.

**Why.** On the evening of 2026-10-16 one session spent its time telling
four others when a pull request had merged, and a new session had to be
asked by message to hold its own; a headless agent reads no message, and a
promise lasts only as long as the session that made it. Since 2026-10-03 a
second person works on the project from another computer, and a cloud
session shares nothing with either but the remote; the product thinker
chose then that the way the project's own people reserve work is a claim
on the shared remote, visible to all of them. The coordination intent's
decomposition, confirmed by the product thinker on 2026-10-17, routed this
stance to a principle.

**Bounds.**

- It governs facts agents rely on from each other. A question put to a
  person, such as the blocker sweep before an unattended run, may travel by
  message, and its answer lands in each session's own checklist
  (dcn-2610180720427437).
- A session's own detail (its checklist, its next step) stays on its own
  computer; only the short line other sessions rely on is published.
- An outside contributor reads the records but cannot write one; the rule
  binds the project's own participants.
- Composes with [enforcement-claims-are-facts](enforcement-claims-are-facts.md):
  a state shown to a person follows what the remote and the machine can
  see, with what an agent says about itself as a hint.

**Live instances.** Reservations (itd-2609091034175565) put who holds what
on the remote; announcements (itd-2609151838312703) put a session's stage
there; the coordination intent (itd-2610170634394993) adds the dependency
line, the shared budget and the central view.

**Promotion.** Once the coordination pass refuses to act on a hold or a
wait that has no record behind it, the rule is enforced and becomes a
discipline.
