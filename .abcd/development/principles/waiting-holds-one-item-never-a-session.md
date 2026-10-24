# Waiting holds one item, never a session

**The rule.** When a piece of work cannot go on until something outside it
happens (a person's answer, another change landing, a credential being
renewed, a slot coming free), only that piece waits. The session holding it
records what the item waits for and when it will look again, then works on
the next item it can do. A session is never idle because one of its items
is. Two consequences follow:

- **A dependency holds only its own step.** A branch that waits on another
  change is held only from being queued: its fixes, its review, its merge
  from the default branch and its preflight go ahead now, and only the
  landing waits.
- **Waiting is a state, not a stop.** A session whose items all wait on
  something outside says so, names each wait, and sets when it next looks,
  so it reads as waiting, never as stopped, and it comes back on its own.

**Why.** On the night of 2026-10-16 one session sat idle for about ten of
eleven and a half hours while it held work it could do alone, and the
coordinating session held three finished, reviewed steps behind a merge
order that mattered only when they landed (iss-2610170619247185). Each
wait was real, and each held far more than the thing waited for. On
2026-10-22 the product thinker ruled that this be written up as a standing
principle.

**Bounds.**

- It governs how work is held, not whether a question is asked: a question
  only the person can answer still waits for them, and only the item it
  belongs to stops.
- It does not make a session start work it is not allowed to do while it
  waits; what it may decide alone is set by the person's standing
  instructions and the away mode.
- An item that can never be unblocked (a dependency closed without merging,
  a cycle) is not a wait: it is refused and named, so nothing waits on it.
- Composes with [enforcement-claims-are-facts](enforcement-claims-are-facts.md):
  the state shown for a waiting session follows what the machine can see,
  not what the session says about itself.

**Live instances.** The stall fix's stop check (itd-2610180912189125) counts
a stacked branch with an open base as owed work, and the coordination
intent's decision that a declared dependency constrains only the landing
(dcn-2610180905291297). The checklist intent (itd-2610170638184486) gives
every item its own wait.

**Promotion.** The stop check is the first mechanical gate on half of this
rule; once the checklist's per-item waits ship, the whole of it is enforced
and it becomes a discipline.
