---
schema_version: 1
id: "iss-2609251639263391"
slug: "the-identity-matchers-do-not-read-the-json-solidus-escape-a"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/identity.go"
deferred_after: "v0.10.0"
deferral_reason: "Judged theoretical in the scanner-cluster fix round: none of the encoders that write abcd's input (Go encoding/json, JSON.stringify for transcripts, Python json) emits the solidus escape, so no transcript, capture or payload abcd reads carries the shape today."
---

The identity matchers do not read the JSON solidus escape. A home path written with escaped forward slashes (\/Users\/LOGIN\/Desktop, \/home\/OTHER\/) raises neither home_path_self nor home_path_other, and a login on the generic list in it raises no local_username; a named login is still caught by its bare word. Only PHP json_encode and org.json write the escape; none of the encoders that feed abcd (Go encoding/json, JSON.stringify for transcripts, Python json) does.
