# Guidance carries its evidence and its purpose

**The rule.** Agent guidance says what it rests on and what it is for. An
instruction that walks a person through a third party's interface carries a
verification tag: observed on screen in this session, or taken from the
vendor's documentation and unverified. Where the person is already in front of
that interface, the agent asks for a screenshot before its second guess, never
after its fourth, and where it can read the page itself it does that first. And
every redaction rule states its purpose beside its pattern, so an agent can tell
an authenticator, which the rule exists to keep out of its hands, from an
identifier that grants nothing on its own.

**Why.** An autonomous run in a managed repository on 2026-09-07/08 walked an
operator through a hosting provider's dashboard to create a token, store it in
two forge secrets and run a workflow. The task is mechanically trivial and took
about ten exchanges. The claims split cleanly by source. Four instructions taken
from the vendor's documentation were all wrong, including two fetched from the
vendor's current guide during the session and quoted exactly: the page had been
rebuilt and the prose had not. Four instructions taken from the operator's
screenshots, minutes later, were all right. A fetched document is evidence of
what someone wrote, not of what the page renders today, and because it carries
the felt authority of a primary source the agent stopped looking. The same run
withheld the hosting account's identifier for four exchanges as a "secret value"
under a standing instruction that listed it beside the token, although it grants
nothing alone, was already public in the repository's own check links, and sat
in the operator's address bar throughout. Every URL the agent gave therefore
arrived as a template with a placeholder, which is what made them unusable. The
rule protected nobody and cost most of the confusion, and because the cost
landed as bad instructions rather than as a refusal, nothing flagged it. The
product thinker adopted both halves on 2026-09-23 (iss-2609100506256173).

**Bounds.**

- The tag describes the source, not the agent's confidence. "From vendor docs,
  unverified" is the honest label for a fetched, accurately quoted guide.
- It governs guidance about an interface abcd cannot observe. A claim about
  abcd's own behaviour answers to
  [enforcement-claims-are-facts](enforcement-claims-are-facts.md) and is checked
  against the code instead.
- A stated purpose never loosens a redaction rule on its own authority. It lets
  an agent see when an identifier sits outside what the rule is for and say so;
  a secret is still never echoed because the purpose seemed not to cover it.
- A runbook step established by observation is worth committing because it
  rots: its value is the date it was seen, and it is re-verified before it is
  trusted again.

**What would show it wrong.** Doc-sourced navigation instructions landing about
four times in five across a handful of vendors would make the tag ceremony, and
it would be dropped.

**Promotion.** The enabling convention is this page and its entry in
`.abcd/work/DECISIONS.md` (2026-09-23). No rung above it exists: nothing checks
that a redaction rule carries a purpose or that a runbook step carries a tag.
The next rung is a purpose field on the redaction rules a managed repository
declares, refused when empty.
