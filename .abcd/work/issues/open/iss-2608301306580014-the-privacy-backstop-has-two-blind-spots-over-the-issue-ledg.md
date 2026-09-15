---
schema_version: 1
id: "iss-2608301306580014"
slug: "the-privacy-backstop-has-two-blind-spots-over-the-issue-ledg"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "itd-179-round-3-security"
found_at: "internal/core/lint"
---

the privacy backstop has two blind spots over the issue ledger: the escaped Windows home path yamlScalar writes and a harness_leak root that excludes the ledger

Found by the round-3 adversarial security review of build/itd-179. Both blind
spots REPRODUCE ON MAIN (verified there for `wontfix_reason` and `resolution`)
and are recorded rather than chased.

Measured: what actually reaches a committed grounds scalar. Redacted:
`/Users/<n>` and `/home/<n>`. NOT redacted: a third-party name, a third-party
email, a corporate hostname, a session URL, an AWS key, and
`C:\Users\<n>\`.

Blind spot 1 -- the escaping defeats the pattern. `abcd lint`'s privacy-hygiene
rule walks all tracked files and DOES catch the raw Windows home form
`C:\Users\alice\...`. It misses `C:\\Users\\alice\\...` -- which is exactly the
form `yamlScalar` writes, because the serialiser escapes the backslashes on the
way in. So the backstop catches the shape a human types and misses the shape
the tool produces. The write-time redactor has no Windows-home pattern at all.

Blind spot 2 -- the root excludes the store. record-lint's `harness_leak` rule
is rooted at `.abcd/development` only, so it never sees `.abcd/work/issues/`
at all. The ledger is outside the reach of that rule by configuration.

Neither is introduced by itd-179, but the branch RAISES THE VOLUME of operator
free text flowing into that store, which is what makes them worth naming now
rather than later. Sibling records: iss-2608301206073609 (the record-write
boundary does not call termsafe.EncodeHiddenRunes) is the same family --
committed record bytes are not held to the standard the repo's own primitives
already implement.

Also from the same review, folded here rather than given its own id: U+007F
(DEL) and the C1/bidi range (U+202E RLO, U+2066, U+FEFF mid-string) pass both
`ValidateText` and `yamlScalar` into committed records, because both gate on
`r < 0x20`. Neither breaks YAML or Markdown parsing; the bidi one is a
Trojan-Source-shaped display concern on committed prose. Pre-existing
serialiser contract, unchanged by this branch. Refusal messages use %q, which
Go escapes, so an ANSI escape in grounds can never reach a terminal raw
through a refusal.
