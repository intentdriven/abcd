---
schema_version: 1
id: "iss-2608311230116767"
slug: "the-cold-reading-ingest-verb-walks-and-deletes-in-the-ledger"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
resolution: "Every path inside the repository is resolved through an os.Root opened at the repository root, and a directory the sweep walks or removes from is additionally refused when it is a symlink."
impact: fix
resolved_by:
  intent: "itd-185"
  spec: "spc-63"
---

The cold-reading ingest verb walks and deletes in the ledger and the durable readings tree with no containment against a symlinked ancestor. rollbackRun does os.ReadDir and os.Remove on .abcd/work/issues/readings/<run> and .abcd/development/readings/<run>, and writeJSON does os.MkdirAll plus WriteFileAtomic under .abcd/development/readings, so a committed git mode-120000 symlink at either path redirects the delete and the write outside the repository root. The sweep runs as step one of Ingest, before the payload is even read, so no valid payload is needed. capture.refuseSymlinkedDir guards the identical ledger directory one file away, and fsutil provides os.Root-scoped primitives for exactly this.

## Grounds

- pursued: five containment cases plant the symlink a hostile clone commits and assert the directory outside the repository is untouched; a later run finding a write or a delete outside would show this wrong
