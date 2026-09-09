---
schema_version: 1
id: "iss-2609091911060345"
slug: "ingest-never-checks-that-the-destination-s-repository-root-a"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "fidelity audit of the recovery intent"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/history/ingest.go"
---

Ingest never checks that the destination's repository root and its store key name the same repository, so the seam that exists to keep one repository's transcripts out of another's corpus does not defend its own invariant. The destination is a pair: a repository root, from which the redaction scanner is built, and a root-commit key, which selects the store the records land in. Ingest refuses an empty root and shape-checks the key, and then trusts that the two describe the same repository. A caller passing a mismatched pair would redact a transcript under one repository's configuration and file it into another's corpus, which is exactly the fault the explicit-destination design was introduced to make impossible. No operator can reach it today, because the only front door derives both halves from a single detection, so this is a latent defect rather than a live one. That is also the reason to close it now rather than later: the argument for the seam is that a destination must never be inferred, and the seam currently relies on its one caller inferring both halves correctly. A second caller, in core or in a future surface, reopens the fault silently. The check is cheap: resolve the root's own root-commit and refuse when it differs from the key.
