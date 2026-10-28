---
schema_version: 1
id: "iss-2610262045024914"
slug: "itd-138-s-install-script-ends-by-running"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1247 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-138-bob-installs-abcd-with-one-checksum.md:25"
remedy: "none (filed automatically)"
related_intents: [itd-138, itd-2609212130136102]
finding_ref: cfk-25834e32a894ccde
---

itd-138's install script ends by running abcd version; itd-2609212130136102 removes that verb in favour of --version

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-138-bob-installs-abcd-with-one-checksum.md:25` — "and finish with `abcd version`"
- End B: `.abcd/development/intents/shipped/itd-2609212130136102-abcd-s-verbs-consolidate-ahoy-s-three.md:40` — "replaces `abcd version`; `version --check` becomes `update --check`"

itd-138 says the install script and the README one-liner, rendered from one template, finish with abcd version. itd-2609212130136102 replaces abcd version with --version and moves version --check to update --check. The install step itd-138 describes ends on a verb the other record removes, so the two cannot both describe the current install path.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1247.
