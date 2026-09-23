# At every refusal, emit what the tool knows

**The rule.** When abcd refuses, the refusal carries what the tool already
holds about the thing it is refusing: the allowed values it validated against,
the value it was given, and the condition it checked. A refusal that names only
the offending input makes the operator rediscover what the tool knew at the
moment it said no.

**Why.** An autonomous run in a managed repository on 2026-09-09/10 found the
centre sound and the failures at the edges. The record store, the
folder-as-status model and the verbs that move records held under 27 parallel
branches with no merge conflict, and the defects that run did find shared one
shape: abcd possessed exactly the information the operator needed, at exactly
the moment it refused, and emitted a refusal without it. It rejected a
`--category` while holding the enum it rejected it against; it asked the host
for a delivered range it had held at `spec close`; it named an include path
without the condition it was checking. The claim was graded and adopted by the
product thinker on 2026-09-23 (iss-2609100509531349). If it holds, the class has
one remedy rather than many, and that remedy also closes the instances nobody
has hit yet.

**Breaches on record.** Each is an open finding of this shape:

- iss-2608290810037524: an unrecognised `--source` value is refused without the
  accepted values.
- iss-2609100506265392: the audit request asks the host for a range the tool
  held at `spec close`.
- iss-2608270559313719: `launch --dry-run` names the include file but not the
  condition, how to create one, or whether this repository needs one.
- iss-2609100508570527: capture does not say that the record it wrote is
  untracked.
- iss-2609100508566033: abcd does not name its own adjacent capabilities.

**Bounds.**

- It governs what abcd already knows. A refusal is not an obligation to go
  and find out more; it is an obligation not to withhold.
- It never loosens a refusal. The refusal stands, and only its message grows.
- A secret, or a private name the banlist guards, is never echoed back because
  it was the refused value. What the tool knows is emitted within the privacy
  rules, not around them.
- The allowed set is named from its one canonical definition, never from a
  second copy written into the message, so the message cannot drift from the
  check.

**Promotion.** A test pattern that drives each refusal path and asserts that
the message carries the allowed set, the given value and the checked condition,
together with a lint that finds refusal paths without such a test, would
promote this to a discipline. Related: `guards-prove-themselves` (a refusal is
tested), `unrecognized-input-never-writes` (what is refused).
