# Guards prove themselves

**The rule.** Any code path whose job is refusal — secret scan, deny rule,
SSRF check, symlink guard, quotation budget, licence detection — ships with a
test that watches it refuse. An untested guard is decoration, and a false
sense of guard is a liability.

**Why.** A guard fails silent: when ordinary code regresses, someone notices
the missing behaviour; when a guard regresses, the system keeps working and
simply stops refusing, which is invisible until the thing it guarded against
happens. The 2026-07-08 review found the launch bundle's symlink-dereference
and scripts-deny guards and memory's quotation-budget and licence-detection
compliance checks all at zero coverage — precisely the paths where a silent
regression costs most. A sibling project enumerates each non-negotiable
invariant alongside a paired test and a mandatory review trigger, which is the
mature form of this rule.

**Instance of a general rule.** This is a special case of [itd-195](../intents/disciplines/itd-195-a-claim-about-how-the-code-behaves-is-executable-or-it-is-no.md), adopted 2026-08-31: a claim about how the code behaves is executable, or it is not made. Stub pointer only — the relationship is recorded, not yet worked through.

**Bounds.**

- The test exercises the *refusal*, not just the happy path around it: it
  presents the forbidden input and asserts the rejection, its error shape, and
  that no side effect occurred. The assertion phase is itself side-effect-free:
  the probe only observes — a check that mutates state on the way to its
  verdict can manufacture or mask the very condition it asserts.
- The proof is bidirectional. The guard's acceptance corpus carries negative
  (`ok:`) cases — permitted inputs it must wave through — asserted alongside
  the refusals, and any change to the guard reruns both sides; a change that
  fixes the refusal side while regressing the permitted side is rejected
  (Weng's [harness-engineering essay](https://lilianweng.github.io/posts/2026-07-04-harness/),
  Lil'Log, 2026, gives this as the
  must-flag/must-pass corpus pair). A guard proven only against forbidden
  input may simply refuse everything, and that failure is as silent as the
  one this rule exists to catch.
- The corpus is curated as key examples — minimal, one distinct facet per
  case, boundary cases included — never every found instance dumped in
  (Adzic's key-examples discipline). A bloated corpus obscures which
  behaviour each case pins and makes the negative side unmaintainable.
- Applies with full force to guards that exist for compliance or trust
  reasons even when no exploit is obvious — those are the ones nobody
  re-checks by hand.

**Promotion.** Pairing each declared invariant with a named test (and a lint
that checks the pairing) promotes this to a discipline; the same lint
requiring at least one negative (`ok:`) case per guard corpus makes the
bidirectionality checkable.
