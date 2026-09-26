---
schema_version: 1
id: "iss-2609252251311346"
slug: "jsonstrict-noduplicatekeys-compares-object-keys-byte-exactly"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/jsonstrict/jsonstrict.go"
resolution: "NoDuplicateKeys folds keys per object the way encoding/json binds struct fields (Unicode simple folding, after unescaping), so a case twin of a receipt's verdict or of rules.json's kill switch is refused by name; jsonstrict carries its own test file and both callers carry the regression."
impact: fix
resolved_by:
  commit: "ff1288f6"
---

jsonstrict.NoDuplicateKeys compares object keys byte-exactly, but encoding/json binds a struct field case-insensitively (its foldName: Unicode simple folding) and keeps the last match. So a receipt carrying "verificationResult": "REJECT" then "VerificationResult": "PROMOTE" passes the duplicate-key refusal and the receipt_gate reads PROMOTE, the exact reviewer-reads-REJECT, gate-reads-PROMOTE evasion the refusal exists to close; "disabled":false,"Disabled":true flips the rules.json kill switch the same way. The check must fold keys per object the way encoding/json matches them.

## Grounds

- pursued: every key spelling encoding/json binds to one field is refused as a duplicate; a spelling the decoder binds that NoDuplicateKeys admits (TestFoldingMatchesEncodingJSONAndEqualFold's premise failing, or a twin passing receipt_gate) would show it wrong
