---
schema_version: 1
id: "iss-2609290300313698"
slug: "the-credential-index-can-split-under-a-same-uid-swap-of-abcd-between-set-s-walk-and-readindex"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/credential/store.go"
deferred_after: "v0.11.1"
deferral_reason: "Deferred out loud by autonomous run A (re-merge of integ/land-14, 2026-09-29), on review-integ14's recommendation: names, homes and pointers only, never a value, and the same uid can edit either index directly, so it is not a gate before the cut. The fix is an exported in-root declaration read, ReadHomeDeclarationDenying over the held *os.Root, used by readIndex under the index lock, with SetMachine taking Set's root for symmetry. capture defer refuses a minor record, so these two fields were set by hand (the iss-2609281654467661 precedent)."
resolution: "Set, SetMachine and the oracle setup's write of ~/.abcd/config.json each read the file they rewrite through the ~/.abcd their walk opened and write through, via the new fsutil.ReadHomeDeclarationDenyingIn, instead of walking ~/.abcd again under the lock. A same-uid swap of ~/.abcd between the walks no longer carries one directory's entries into the other. The review rationale holds (names and pointers only in the index, and the same uid can edit either file), but the store and the config read had the same shape and the store's does carry values, so the fix was taken rather than the deferral kept."
impact: fix
resolved_by:
  commit: "6599f17a0"
---

The credential index can split between two ~/.abcd directories under a same-uid swap of one real directory for another between Set's walk and the index's re-walk. Set opens ~/.abcd once by descriptor and writes the index through that held root (store.go setIndex, WriteFileAtomicInRoot), but readIndex under the index lock re-walks ~/.abcd by path (fsutil.ReadHomeDeclarationDenying). If ~/.abcd is renamed aside and a different real directory swapped in between the two walks, the index read comes from the new directory while the write goes to the held one: the held directory's index becomes the new one's entries plus the new name (its own entries lost), and the live ~/.abcd does not record the new name. Names, homes and pointers only, never a value: SetMachine re-walks for its read and its write, so the value and its store stay together in the live directory. Same uid, who can edit either file directly. Surfaced by the security review of integ/land-14 (review-integ14 item 4, LOW (b)).

## Grounds

- pursued: a read-modify-write under a home-scoped lock reads and writes one directory; shown wrong if TestSetKeepsTheIndexInTheDirectoryItHolds, TestSetMachineKeepsTheStoreInTheDirectoryItHolds or TestTheProviderBlockIsReadWhereItIsWritten sees an entry cross between the held and the swapped-in directory
