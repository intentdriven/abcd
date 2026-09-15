# Basics built in, an external dependency brings full power

**The rule.** Every abcd capability ships with a basic form that works with
no daemon and no dependency beyond the abcd binary and the operating system,
and offers an opt-in adapter to an external dependency for the full form. The
basic is never a stub: it does the job at the scale a single person or a small
team hits first, and it keeps working when the adapter is absent. The adapter
never becomes a precondition: a repository or a machine that declines it
loses reach or speed, not the capability.

**Why.** abcd is for people who know what they want to build and need help
shipping it; a capability that needs a broker, a service, or a network before
it does anything is a capability most of them will never switch on, and a
capability that only works through a vendor's cloud is one they cannot trust
with their record. The basic form is also where the contract is discovered,
which is the same reason the [script-first MVP](script-first-mvp.md) puts the
first cut in a script: the on-disk shapes, the failure modes and the verbs
worth having stabilise under real use before the adapter has to preserve
them. And the adapter's cost is real: a running process, an authentication
story, a second audit trail; it should be paid by the people who need what it
buys, not by everyone.

**Bounds.**

- The basic form and the adapter share one record contract. What the basic
  writes as a file, the adapter mirrors as a file; a capability is never
  auditable in one form and opaque in the other.
- The adapter is host-agnostic in the same way the basic is: it depends on
  the external tool, never on a particular agent harness or a vendor's
  cloud. A local-network dependency qualifies; a hosted relay does not.
- "Basic" is measured at the user surface, not at the code: the test is
  whether a person with the binary alone can do the thing, not whether the
  code path is short.

**Live instances.** The oracle backends (host-delegated by default; native,
CLI, API and MCP as opt-in adapters). The secret scanners (the native
detector is the default; gitleaks and trufflehog are opt-in dependencies the
install offers). The session mailbox (a shared-directory mailbox is the
basic; an embedded broker is the adapter).

**Promotion.** This principle is the standing statement of the "host-delegated
by default" boundary in the conventions, widened from LLM work to every
external dependency. No enforcement hook exists; per the promotion path in
this directory's [README](README.md), it becomes a discipline-kind intent
the moment one does.
