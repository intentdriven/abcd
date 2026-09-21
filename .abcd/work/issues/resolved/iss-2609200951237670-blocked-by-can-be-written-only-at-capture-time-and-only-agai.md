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
resolution: "abcd capture link <iss-N> --blocked-by/--unblock writes and removes the blocked_by edge after capture, in any status folder, through the validator the capture flag runs; the refusal for an absent target and the flag help on both verbs name where the field is documented."
impact: additive
resolved_by:
  commit: "994e2be0ce4b9efeaaf7a14f5b5bdad443a8bca1"
---

blocked_by can be written only at capture time, and only against a record that already exists. abcd capture --blocked-by refuses a target absent from the ledger ("not found in the issue ledger; nothing written"), which is right for a typo but leaves no path for the ordinary case: the blocker is captured after the blocked record, or in another lane, and no verb adds the link afterwards (the capture sub-verbs are disposition, list, mentions, promote, resolve and wontfix). The Gropius session gropiusllm-97 hand-wrote blocked_by into frontmatter on 2026-09-20 and found the key's shape by running strings on the binary; the shape is in fact documented (the issue-store README and the capture surface page both carry it), so the discoverability half is that neither the refusal nor the flag help points there. Wanted: a post-hoc link verb, abcd capture link <iss-N> --blocked-by <iss-M> (and its removal), validated the way capture validates the flag, so the edge is written by a verb whichever record came first; and the refusal naming where the field is documented. Sibling of iss-2609091256264547 (the three typed relations the schema lacks): this is the one relation it has and cannot write after the fact.

## Grounds

- pursued: we expect a session whose blocker lands after the blocked record to write the edge by verb rather than by hand-editing frontmatter, and a refusal that names the documentation to end the strings-on-the-binary discovery; a later record of a hand-written blocked_by, or of a session that could not find the field's shape, would show it wrong
