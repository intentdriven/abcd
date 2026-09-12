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
resolution: "Ingest now verifies its destination. A new check resolves the destination root's own root commit through the same seam the store location uses and refuses when it disagrees with the supplied store key, so the pair can no longer name two different repositories. It fails closed when the root's own root commit does not resolve at all, because an unresolvable root is not evidence that the pair agrees. Ten existing test tables were passing mismatched pairs, exactly the shape the check now refuses, and each now describes one repository."
impact: fix
resolved_by:
  intent: "itd-2609091718566731"
  spec: "spc-2609091722230648"
---

Ingest never checks that the destination's repository root and its store key name the same repository, so the seam that exists to keep one repository's transcripts out of another's corpus does not defend its own invariant. The destination is a pair: a repository root, from which the redaction scanner is built, and a root-commit key, which selects the store the records land in. Ingest refuses an empty root and shape-checks the key, and then trusts that the two describe the same repository. A caller passing a mismatched pair would redact a transcript under one repository's configuration and file it into another's corpus, which is exactly the fault the explicit-destination design was introduced to make impossible. No operator can reach it today, because the only front door derives both halves from a single detection, so this is a latent defect rather than a live one. That is also the reason to close it now rather than later: the argument for the seam is that a destination must never be inferred, and the seam currently relies on its one caller inferring both halves correctly. A second caller, in core or in a future surface, reopens the fault silently. The check is cheap: resolve the root's own root-commit and refuse when it differs from the key.

## Grounds

- pursued: we expect resolving the root's own root commit to be a complete check because the store key IS a root commit, so the two are directly comparable and no heuristic is involved; it is shown wrong if a legitimate caller must address a store whose key is not its root's own commit, which would mean the key means something else
