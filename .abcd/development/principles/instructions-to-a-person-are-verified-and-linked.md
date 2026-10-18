# Instructions to a person are verified and linked

**The rule.** When an agent tells the product thinker or the technical
facilitator to do something outside the repository (change a forge setting,
create a key, register an app, publish an advisory, start a hosted session),
it first checks the current state of that service against its own
documentation, then gives a direct link to the page where the person makes
the change. Two parts, both required:

- **Verified now.** Menu paths, page names, URLs and defaults move. An
  instruction written from memory describes the service as it was when the
  agent learned it, so the agent searches or reads the service's own
  documentation in the same session, before the instruction is given, and
  reads the live state it can reach itself (the repository's current
  rulesets, the advisory's state) so the instruction starts from what is
  there.
- **Linked.** Each step that happens on a website names the exact page as a
  full URL, filled in for this repository or organisation (for example
  `https://github.com/<owner>/<repo>/settings/rules`, not "go to Settings,
  then Rules"). A menu path may follow the link as a fallback, never replace
  it.

A link the agent could not confirm is marked as unconfirmed, beside the page
or document that names it, so the person knows which link may have moved.

**Why.** The person reads these instructions away from the session, often in a
hurry and between other work, and acts on them alone. A menu path from memory
costs them a search the agent could have done, and a stale one sends them to a
page that no longer exists. On 2026-10-17 the product thinker, given a list of
web-only steps before a week away, asked for the URLs and ruled that every
such instruction checks its facts first and links to the place the change is
made.

**Bounds.**

- Applies to instructions a person carries out. Explaining what a setting
  does, without asking anyone to change it, needs neither part.
- A link never carries a secret, a token, or a pre-filled credential.
- Composes with [prefer-sota](prefer-sota.md) (check outside practice
  against primary sources) and
  [guidance-carries-its-evidence-and-its-purpose](guidance-carries-its-evidence-and-its-purpose.md).
