---
schema_version: 1
id: "iss-2609012036271396"
slug: "ghsa-fh9j-8xmg-m33f-cwe-59-cwe-400-the-issue-ledger-readers"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/workflow.go"
resolution: "Both issue readers now go through readRecordGuarded (moved to roots.go beside readWithChecksum): scanLedger turns a FIFO, device, symlinked or oversize leaf into a SkipRecord carrying ErrPathUnsafe, and readWithChecksum refuses the same leaves, so Promote, transition and commitCapture inherit the guard. The cap is issueschema.RecordReadLimit, now documented as the issue family's too. TestScanLedgerRefusesNonRegularOversizeAndSymlinkedRecords proves List and Status return within a deadline on a FIFO, skip an oversize record, and skip a symlinked leaf without serializing its target; TestPromoteRefusesASymlinkedRecordLeaf proves the pre-flight read refuses a symlinked record with nothing minted."
impact: fix
---

GHSA-fh9j-8xmg-m33f (CWE-59, CWE-400): the issue-ledger readers scanLedger (workflow.go) and readWithChecksum (roots.go) load every well-formed iss-N-*.md through a bare os.ReadFile, with no O_NOFOLLOW, no O_NONBLOCK, no regular-file check and no byte cap, and List/Status never pass mutationPreamble so the write path's leaf guards never run on a read. In a hostile clone a committed FIFO at a record name hangs capture list and the bare capture status render, an oversize record is read and serialized unbounded, and a committed symlink to an out-of-tree schema-shaped file has its body (any secret span included) serialized into list --json, with no redactor on the read path. The package already owns the hardened reader (reading.go readRecordGuarded, fsutil.ReadGuarded under issueschema.RecordReadLimit) and the reading family uses it everywhere; the issue family was never swept. The fix must route both readers through it so a FIFO, device, symlink or oversize leaf becomes a SkipRecord the surfaces already render, and Promote, transition and commitCapture inherit the guard through readWithChecksum. Sibling unguarded reads outside this package are captured as their own records.

## Grounds

- pursued: the hardened reader already existed in this package and was used by every other record family, so the fix is routing the two issue readers through it rather than a second reader or a scan-time redactor — one primitive, one cap, one refusal text for every family
