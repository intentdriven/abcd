---
schema_version: 1
id: "iss-2610251558404922"
slug: "docs-explanation-artefacts-md-19-says-an"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "docs-currency review of docs/, 2026-10-24, reader D2"
origin: researcher-authored
production_mode: hand-written
found_at: "docs/explanation/artefacts.md"
remedy: "Say the intent is filed as shipped when its spec closes, that the build loop audits before that close, and that an audit finding a criterion unmet leaves the intent shipped and flagged until the fix is re-checked."
---

docs/explanation/artefacts.md:19 says an intent is filed as shipped once its acceptance criteria are verifiably met. That holds on the build loop's closing lane, where an unmet or undecided audit fails the round (internal/core/implement/loop/validate.go:129,186-205). A spec closed by hand ships the intent first and the audit follows; an unmet criterion then leaves the intent in shipped/ with an owed check and an issue (internal/core/intent/audit_owed.go:9-21, ruling DQ1c).
