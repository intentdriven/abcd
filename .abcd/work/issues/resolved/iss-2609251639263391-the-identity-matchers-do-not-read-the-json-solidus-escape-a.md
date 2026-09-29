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
resolution: "Fixed by c55ae5ef: the JSON-escape views decode \\/ and \\u002f (either case) to '/', so a solidus-escaped home raises home_path_self or home_path_other and a generic login's escaped home is the caller's home. Escape spellings swept and decided: \\/ and \\u002f match (JSON views); %2F matches (the percent pre-pass, pre-existing); a doubled or quadrupled backslash separator matches (the Windows alternative reads separator runs, iss-2609251639261103); \\x2f is not matched, because no encoder that feeds abcd writes '/' that way (a Python repr or C escape only escapes non-printables); an HTML entity (&#47;, &#x2F;) is not matched, because abcd's inputs are JSON and markdown and an entity-escaped path in a fetched page is not a spelling the store receives today. Pinned by TestJSONSolidusEscapeReadsAsASeparator, watched RED at 211b8853."
impact: fix
resolved_by:
  commit: "c55ae5ef"
---

The identity matchers do not read the JSON solidus escape. A home path written with escaped forward slashes (\/Users\/LOGIN\/Desktop, \/home\/OTHER\/) raises neither home_path_self nor home_path_other, and a login on the generic list in it raises no local_username; a named login is still caught by its bare word. Only PHP json_encode and org.json write the escape; none of the encoders that feed abcd (Go encoding/json, JSON.stringify for transcripts, Python json) does. <!-- abcd-lint:allow -->

## Grounds

- pursued: a home path spelled with the JSON solidus escape is found and masked wherever ScanText runs; an \/ or \u002f spelled home reaching the store verbatim would show it wrong
