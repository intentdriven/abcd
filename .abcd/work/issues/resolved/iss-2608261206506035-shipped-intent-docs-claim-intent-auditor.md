---
schema_version: 1
id: "iss-2608261206506035"
slug: "shipped-intent-docs-claim-intent-auditor"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "bughunt-a/round-8"
found_at: ".abcd/development/intents/README.md"
resolution: "shipped/ front doors now say the close hook moves the intent and emits an OWED receipt the auditor replaces later, not that the auditor gates entry."
impact: internal
resolved_by:
  commit: "1ec3719f"
---

shipped intent docs claim intent-auditor ran as an entry precondition it does not gate