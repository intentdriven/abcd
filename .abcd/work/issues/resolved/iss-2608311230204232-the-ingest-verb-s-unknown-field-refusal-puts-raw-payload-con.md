---
schema_version: 1
id: "iss-2608311230204232"
slug: "the-ingest-verb-s-unknown-field-refusal-puts-raw-payload-con"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
resolution: "The unknown-key list goes through the same cleaner and caps as every other echoed value, per name and on the number of names."
impact: fix
resolved_by:
  intent: "itd-185"
  spec: "spc-63"
---

The ingest verb's unknown-field refusal puts raw payload-controlled JSON key names into ItemRefusal.Field without echo(), so they reach the committed run.json and the --json render un-sanitised and un-capped. encoding/json escapes C0 but not C1, DEL, bidi overrides or zero-width runes, which internal/termsafe/json_test.go already pins. The sibling Detail field goes through renderFields and is clean, and ingest.go's header claims every payload-derived string reaching a terminal or a durable record is sanitised and capped.

## Grounds

- pursued: the finding is closed by a test that fails without the change; a later review or mutation run finding the same shape again would show this wrong
