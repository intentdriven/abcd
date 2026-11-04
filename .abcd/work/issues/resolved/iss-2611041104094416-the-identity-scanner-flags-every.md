---
schema_version: 1
id: "iss-2611041104094416"
slug: "the-identity-scanner-flags-every"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "review of the cloud session's pull request #1384, 2026-11-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/identity.go"
remedy: "Treat the account name as one that identifies no person, beside the other default accounts of hosted machines."
resolution: "The account name joins the built-in list of names that identify no person, and the plugin's own install line is spared where the name stands before an at sign."
impact: fix
---

The identity scanner flags every word-bounded occurrence of the caller's account name, and abcd's cloud setup names its account abcd, so record text holding the product's name read back as [redacted-user] and launch renders hard-failed on a clean tree in a cloud session.
