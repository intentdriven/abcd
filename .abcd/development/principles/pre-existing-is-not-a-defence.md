# Pre-existing is not a defence

**The rule.** A defect confirmed while doing other work is not a discovery to
file and move on from. The moment it is confirmed there are exactly two moves:
fix it, or defer it out loud. A deferral is itself a decision, recorded where
the release can see it, naming the finding and the reason; it is never a silence
that happens to look like a decision afterwards. That a defect was already there
before the work began settles nothing about what to do next. It is a fact about
the defect's age, and the question on the table is about its consequence.

On a security or bug-fix release the second move narrows. A confirmed finding of
consequence is fixed in that release rather than carried past it, because a
release that steps over a known defect of exactly the class it is named for
spends the credibility it exists to build. The users who read a security release
note and act on it are trusting a claim about the state of the system; a defect
the maintainers had already confirmed and stepped over makes that claim false in
the specific way that is hardest to recover from.

**Why.** The failure has a shape, and the shape is persuasive rather than
careless. Nobody proposes to ignore a defect. What gets proposed is to *record*
it: capture the finding, cite it in the release notes, keep the cut moving,
handle it next cycle. Every step of that is a real practice this repository
already runs, which is what makes the move so easy to make and so hard to
notice. What it quietly does is convert "this is broken" into "this is filed",
and filing is not a decision about the defect. It is a decision to make no
decision, wearing the record's clothes.

Measured here: during the preparation of the release following v0.7.1, that
exact proposal was made three times against confirmed findings, on the grounds
each time that the defect predated the work in hand. It was overruled three
times. Three of one shape in one cycle is not three mistakes; it is a rule the
record did not yet state, so nothing could be checked against it.

**The tension, stated.** A principle that pretends the cost is zero is worth
nothing, because the first person to meet the cost will conclude the principle
was written by someone who had not. Fixing everything a cycle turns up is
unbounded: adversarial review produces findings faster than fixes land, and a
release that never cuts protects nobody. So the rule is deliberately not "fix
every finding". It is two claims, and only the second is about fixing:

- No finding passes unrecorded. This one is absolute and cheap.
- No consequential finding passes unfixed, in a release about that class of
  defect, without a decision that says so out loud.

The escape is real and it is meant to be used. What it is not is free: a
deferral costs a sentence naming the finding and the reason, in a place the next
release will read back. That price is the whole mechanism. It is low enough that
a genuine deferral is never blocked, and high enough that a deferral cannot be
made by saying nothing.

**Bounds.**

- Consequence is the author's own grade, not a second opinion the gate forms.
  A finding graded `major` or `critical` is one its author called consequential,
  and the gate takes that at face value. The remedy for a grade that overstates
  is to re-grade it honestly in a reviewed change, never to argue past it.
- A grade that cannot be read counts as consequential. Not judged must not read
  as not serious: the failure this whole rule exists to close is a finding
  passing because nobody looked at it.
- `wontfix` satisfies the rule completely. A recorded decision not to act, with
  its reason, is the conscious non-action asked for here; it is the opposite of
  ignoring a finding, not a way around the requirement.
- The standing backlog is out of scope. A finding already in the record when
  the cycle began is not what this cycle found, and a rule that held every
  release hostage to the whole ledger would be switched off rather than
  satisfied.
- Composes with [workaround-records-the-defect](workaround-records-the-defect.md),
  which governs the moment before this one: that rule gets the defect into the
  record, this one decides what the release does about it.
- Composes with [the-record-lands-with-the-act](the-record-lands-with-the-act.md).
  A deferral scheduled for after the cut is the step that gets forgotten, for
  the reason that rule gives, so the deferral is written in the diff that ships
  the release rather than promised alongside it.
- Answers to [enforcement-claims-are-facts](enforcement-claims-are-facts.md).
  This rule is the kind that would be pure exhortation without a gate, and an
  ungated rule about discipline under release pressure is exactly the claim that
  degrades behaviour by being believed. The gate below is what makes the
  paragraphs above description rather than aspiration.
- Related to [loud-staging](loud-staging.md) by inversion. That rule permits
  incomplete work on condition it announces itself; this one permits an unfixed
  finding on the same condition. Both refuse the same thing, which is a system
  that cannot be trusted to explain its own state.
- Related to [fix-the-detector](fix-the-detector.md) at the next rung up. This
  rule binds the individual finding; that one asks what would have caught its
  whole class. A finding fixed under this rule and left with no detector has
  satisfied the release, not the review.

**Live instance.** `internal/core/changelog.GuardFindings` is the armed rung. At
a release cut it takes the set-difference of issue-ledger membership between the
anchor tag's tree and HEAD, keyed on record id, and refuses the cut when any
record that difference names is still in `open/` and graded `major` or
`critical`. The refusal names every blocking record and all three ways out, and it
reaches `abcd changelog` and `abcd launch ship` through the same `refusals`
channel as the surface guardrail. Membership is read from git rather than from
the timestamp inside a record id, so the gate's verdict is not decided by a
field the record it is judging can edit.

The waiver is the `deferred_after` / `deferral_reason` frontmatter pair on the
record, written by `abcd capture defer`. `deferred_after` names the anchor tag the deferral was granted against,
which makes it single-use: when the next release re-anchors, the waiver lapses
and the finding is re-asked. A waiver that names the wrong anchor, or states no
reason, leaves the record blocking and says why, in the same fail-safe direction
`shipped_in` takes. A deferral that stands is carried onto the cut and printed
in the release report, because a deferral nobody can see is indistinguishable
from having ignored the finding.

The first cut to run the gate refused, naming six findings the cycle after
v0.7.1 had captured and left open. That is the correct answer, and it is what
the rule is for.

**Promotion.** The enabling convention is the severity grade the ledger already
carries; the armed rung is the release-cut refusal above. The rung still absent
is the write path: the waiver pair is a hand edit today, with no `abcd capture`
verb behind it, so the one act this rule wants to make easy is the one act with
no tooling. A `capture defer` verb that stamps both fields against the current
anchor is the next step, and until it ships, a hand-edited waiver is the
documented form.
