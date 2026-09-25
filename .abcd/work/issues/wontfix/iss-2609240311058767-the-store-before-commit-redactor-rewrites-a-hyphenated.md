---
schema_version: 1
id: "iss-2609240311058767"
slug: "the-store-before-commit-redactor-rewrites-a-hyphenated"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "autonomous run 2026-09-23 fidelity audit"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/redact.go"
wontfix_reason: "duplicate of iss-2609061504302157: the same redactor rewriting an ordinary word that equals the local account name, observed here in a hyphenated term on the intent audit ingest"
---

The store-before-commit redactor rewrites a hyphenated vocabulary word when its first token equals the local account name. Observed on `abcd intent audit ingest` for itd-4 (receipt rcp-2662745d5344): the verdict's ac-4 rationale named the historical sync verb of the plugin, the two-token hyphenated term the intent's own fourth acceptance criterion carries verbatim, and the ingested Audit Notes now read `[redacted-user]-sync` in two places (the criterion line and the gap-audit claim) while the criterion text forty lines above carries the term unredacted. The identity kinds in internal/adapter/scanner/redact.go (kindLocalUser, replaced by the marker at redactionReplacement) treat a hyphen as a word boundary, so a product term is redacted as a person. The committed record is machine-dependent: on an account with another name the same ingest writes the term. Found during the fidelity audit of itd-4 in the autonomous run.

## Grounds

- declined: the finding is carried whole by iss-2609061504302157; this would be wrong if iss-2609061504302157 were closed without answering it
