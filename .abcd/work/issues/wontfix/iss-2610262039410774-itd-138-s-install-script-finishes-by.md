---
schema_version: 1
id: "iss-2610262039410774"
slug: "itd-138-s-install-script-finishes-by"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 878 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-138-bob-installs-abcd-with-one-checksum.md:25"
remedy: "none (filed automatically)"
related_intents: [itd-138]
finding_ref: cfk-035067be8ec1666a
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The register names the record that made the change, and the template is current."
---

itd-138's install script finishes by running `abcd version`; the surfaces register says `version` is a root flag, not a verb

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-138-bob-installs-abcd-with-one-checksum.md:25` — "print the PATH fix when needed, and finish with `abcd version`"
- End B: `.abcd/development/brief/04-surfaces/README.md:156` — "`version` is the root's `--version` flag (itd-2609212130136102), not a verb"

itd-138 says the install script, rendered from the same template as the README one-liner, finishes with `abcd version`. The surfaces register says `version` is the root's `--version` flag and not a verb, so the final step the intent describes invokes a command that the brief's verb tables do not register.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 878.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The register names the record that made the change, and the template is current.
