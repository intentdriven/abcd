---
schema_version: 1
id: "iss-2611221756106291"
slug: "the-capture-page-lists-five-of-the-six"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "mining the parked phase-9 clone, 2026-11-22"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/capture.md"
remedy: "Add the sixth state to the page's sentence, and to the brief's capture page if it carries the same list; the closed spec stays as written."
---

commands/capture.md lists five of the six states the `record_provenance` rule reports for the disclosure pair. The rule's header in internal/core/lint/provenance.go enumerates six, and the page leaves out the last: a reading pointer naming an item whose own `related_intents` names some other record, the same join read from the item's end. A reader of the page cannot learn every finding the gate can raise. A detection reading of 2026-09-02 found three counts standing at once (a closed spec's four, the rule's six and the page's five); the answer given to that item accepted it and said the page and the rule would be made to agree, and no record then carried that.
