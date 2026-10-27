---
schema_version: 1
id: "iss-2610262034494554"
slug: "itd-138-s-install-script-finishes-by"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 526 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-138-bob-installs-abcd-with-one-checksum.md:25"
remedy: "none (filed automatically)"
related_intents: [itd-138]
finding_ref: cfk-af5ffbc90e5234ad
---

itd-138's install script finishes by running abcd version; the board chapter says version is a retired name answered with its replacement.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-138-bob-installs-abcd-with-one-checksum.md:25` — "print the PATH fix when needed, and finish with `abcd version`"
- End B: `.abcd/development/brief/04-surfaces/08-abcd.md:456` — "A name retired with no alias is answered with what replaced it, and never as a stale binary"

itd-138 says the install script served from the project's domain, rendered from the same template as the README one-liner, finishes with abcd version. 08-abcd says a name retired with no alias, version among them, is answered with what replaced it, the version report now being the root's version flag. An installer whose last step invokes a retired name ends on the refusal of an unknown command rather than on the version report the intent promises.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 526.
