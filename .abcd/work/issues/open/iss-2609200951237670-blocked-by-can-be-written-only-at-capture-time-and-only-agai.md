---
schema_version: 1
id: "iss-2609200951237670"
slug: "blocked-by-can-be-written-only-at-capture-time-and-only-agai"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "Gropius intent lifecycle run, session gropiusllm-97, relayed to abcd-17 on 2026-09-20"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/cli.go"
---

blocked_by can be written only at capture time, and only against a record that already exists. abcd capture --blocked-by refuses a target absent from the ledger ("not found in the issue ledger; nothing written"), which is right for a typo but leaves no path for the ordinary case: the blocker is captured after the blocked record, or in another lane, and no verb adds the link afterwards (the capture sub-verbs are disposition, list, mentions, promote, resolve and wontfix). The Gropius session gropiusllm-97 hand-wrote blocked_by into frontmatter on 2026-09-20 and found the key's shape by running strings on the binary; the shape is in fact documented (the issue-store README and the capture surface page both carry it), so the discoverability half is that neither the refusal nor the flag help points there. Wanted: a post-hoc link verb, abcd capture link <iss-N> --blocked-by <iss-M> (and its removal), validated the way capture validates the flag, so the edge is written by a verb whichever record came first; and the refusal naming where the field is documented. Sibling of iss-2609091256264547 (the three typed relations the schema lacks): this is the one relation it has and cannot write after the fact.
