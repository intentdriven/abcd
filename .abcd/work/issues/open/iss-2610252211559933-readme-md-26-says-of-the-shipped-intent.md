---
schema_version: 1
id: "iss-2610252211559933"
slug: "readme-md-26-says-of-the-shipped-intent"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "docs-currency review of docs/, 2026-10-24, reader D8"
origin: researcher-authored
production_mode: hand-written
found_at: "README.md"
remedy: "Replace 'That verdict is recorded, not enforced.' on README.md:26 with: 'That verdict is recorded rather than gating the intent: a criterion judged not met leaves the intent shipped and files a major issue, which the next release cut does not pass while it stays open.'"
---

README.md:26 says of the shipped-intent audit: 'That verdict is recorded, not enforced.' When a verdict judges a criterion NOT_MET, the ingest files a major bug in the ledger's open/ folder (internal/core/capture/auditowed.go:25-37; commands/intent.md:1009-1015). The release cut refuses to go ahead while a major record captured since the anchor tag is still open (internal/core/changelog/findings.go:62, GuardFindings at :193, called from internal/core/release/emit.go:267). So the verdict does not unship the intent, but a failed one stops the next release until it is fixed, declined with a reason or deferred, and the sentence leaves that out. Judgement call: the sentence may be meant only as the verifier-selects principle, that no model verdict decides admission.
