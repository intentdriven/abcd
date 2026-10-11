# abcd fixes what it adds

**The rule.** Every security finding is assessed with one question first: would
this exist without abcd?

- **No:** abcd added it, so it is abcd's vulnerability. It is fixed, and the fix
  does not change how the repository works.
- **Yes:** it is not abcd's vulnerability. abcd helps prevent it only where it
  can do so without changing how the repository works, for example by noticing
  a change and stopping to say so. Where it cannot, it is not abcd's problem: the
  finding is closed with that reason, not fixed.

A fix never changes how a repository works on its own. Its hooks, its content
filters, its large-file storage, its merge tools and the way it reaches its
remote stay as the repository set them.

**Why.** The product thinker's ruling of 2026-10-11, in their words: "We don't
want to interfere with how non-abcd-managed repos work, but want to make sure
that abcd isn't adding security vulnerabilities and, only if possible, help to
prevent existing ones", and "If an issue would exist without abcd and abcd
can't help preventing it, it's not abcd's problem." It came out of a triage in
which each proposed fix switched off a piece of the repository's own git
configuration to stop a program it names from running, and each one broke
something a real repository relies on (large-file storage, its own hooks, its
login helpers), only to move the risk to the next piece. A test on a scratch
repository settled the question the fixes had skipped: a program named in a
repository's shared git settings runs on the person's own next plain `git add`
or `git commit`, with no abcd involved. Switching it off for abcd alone removed
nothing an attacker could do and broke what the repository's owner had set up.

**Bounds.**

- The question is about the route, not the harm. A risk is abcd's when abcd
  creates the way in: a command abcd runs that reaches something the person's
  own tools would not, a file abcd writes where it should not, a secret abcd
  stores or ships, a check abcd claims and does not perform. The same risk
  reached the same way without abcd is not abcd's.
- abcd's own protections are prevention, and they are kept honest. The guard,
  the release scan and the record gates exist to catch what would happen
  anyway. A gap in one, where closing it changes nothing about how a
  repository works, is fixed: a protection abcd states and does not deliver is
  a check abcd claims and does not perform, which is abcd's.
- Closing is a recorded decision, not a silence.
  [`pre-existing-is-not-a-defence`](pre-existing-is-not-a-defence.md) still
  holds: a finding that is not abcd's problem moves to `wontfix/` (or its
  advisory is closed) with the reason this rule gives, naming why it would exist
  without abcd and why abcd cannot help without changing how the repository
  works.
- Adjacent, not the same:
  [adr-42](../decisions/adrs/0042-guard-parse-layer-is-a-mistake-filter.md) makes
  the guard a mistake filter for a cooperating agent rather than a security
  boundary; this rule says which findings abcd owns at all, across every
  surface.

**Enforcement.** None mechanical. The `SECURITY` domain in `.abcd/rules.json`
puts the test in front of every session whose prompt is about security work,
and `.github/SECURITY.md` states the scope to reporters. A finding's record
names which side of the question it is on.

**Promotion.** A field on a security record saying whether the route exists
without abcd, refused by record-lint when absent, would make the rule checkable
and promote it to a discipline.
