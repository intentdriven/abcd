---
schema_version: 1
id: "iss-2609020239008775"
slug: "the-memory-store-s-write-time-leaf-walk-judges-values-only-s"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/memory/redact.go"
resolution: "redactLeaves now judges an introduced KEY with judgeKey before it judges the value under it, using the same store redactor the values go through: a key carrying a blocking secret or identity span refuses the whole write, and the refusal names the label and the kinds, never the key, because here the key is the secret. Refusal rather than rewriting is the rule — renaming a key renames the field the reader looks up and dropping it discards the value it names, so neither is a redaction. A key the store already holds is not re-judged, the same no-lockout rule the value side has. The doc comment now says keys are not schema-fixed instead of asserting the opposite. TestWriteRefusesASecretShapedKey pins both arms: a ghp_-shaped key in a --pages-json source block refuses the ingest and writes no page, and an ordinary extra key is still written."
impact: fix
---

the memory store's write-time leaf walk judges values only, so a secret can enter as a map KEY. storeRedactor.redactLeaves sanitises every string leaf a write introduces but never looks at the keys it walks, and its comment asserts 'Keys are schema-fixed and never walked' — false for the page source: block, where validateSourceBlock rejects no unknown key and checkBlockKey admits any identifier-shaped key, a token like ghp_ followed by alphanumerics included. A host distiller's --pages-json frontmatter can therefore carry a credential as a YAML key straight into a committed page, and only the MR001 residue lint catches it, on read, after the write has already happened. The fix must judge an introduced key with the same scanner its values go through (refuse, never rename) and correct the comment.

## Grounds

- pursued: judging keys where the values are already judged keeps one detector and one write boundary, rather than adding a second unknown-key rule in validateSourceBlock that would only cover the page source block and not the registry; if a legitimate distiller key ever trips the scanner the refusal will name the kind and show the rule is too tight
