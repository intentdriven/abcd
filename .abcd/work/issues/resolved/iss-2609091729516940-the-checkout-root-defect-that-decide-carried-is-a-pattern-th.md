---
schema_version: 1
id: "iss-2609091729516940"
slug: "the-checkout-root-defect-that-decide-carried-is-a-pattern-th"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "adversarial-review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/cli.go"
resolution: "All six front doors resolve the checkout root before they address a record store, through one promoted primitive rather than six copies: capture, decide, spec, memory, intent and ideate record each refuse outside a repository with nothing read or written, address the checkout's own store from a subdirectory, refuse a repository-shaped tree git will not answer for rather than falling through to the marker walk, and report a store found below the root instead of adding to it."
impact: fix
---

The checkout-root defect that decide carried is a pattern the surface shares, not two instances of it. iss-2609090951291524 fixed it for the seven capture verbs and iss-2609091707224329 for decide; the same raw shape — os.Getwd() handed to a core that joins a repo-relative store path onto whatever it is given — remains at the intent, spec, ideate and memory front doors, which between them own the intent store, the spec store, the ideate verdict record and the memory substrate. Reproduced against a scratch git checkout holding one intent draft: abcd intent run from internal/core reports drafts 0, planned 0, shipped 0 while the identical binary at the root reports the draft, and abcd intent "<text>" run from the same directory mints a second store at internal/core/.abcd/development/intents/drafts and reports a repo-relative path that reads exactly like the checkout store's. Run in a plain directory that is not a repository at all it exits 0 and lays the full intent skeleton there. spec and memory show the read half of the same thing from a subdirectory. The cost is the one the two closed records already argue: a record filed where no gate, no release cut and no reader looks, written with every appearance of success, and for the intent store an intent whose spec can never be closed against it. Fix direction: the resolution is already shared and already generalised — gitutil.CheckoutRoot takes git's toplevel, refuses a repo-shaped tree git will not answer for, and refuses when nothing repo-shaped sits above, with the store's noun as its only parameter — so each remaining front door needs a resolving step of the captureLedgerRoot and decideStoreRoot shape and the strayStoreNotes call that reports what an unresolved door already deposited, not a new resolver. Detector, per family: the verb run from a subdirectory addresses the checkout's store, the verb run outside a repository refuses with exit 2 and writes nothing, the verb run at the root is unchanged, and a static pass over the surface package refuses a request built from an unresolved working directory.

Two of the four families are closed on 2026-09-09: `spec` and `memory` now
resolve the checkout root through gitutil.CheckoutRoot before they build a
request, at all six call sites (bare `spec`, `spec close`, bare `memory`,
`memory ingest`, `memory ask`, `memory lint`), each with the family's own
detectors and the strayStoreNotes report. The record stays open for `intent` and
`ideate record`, which are a sibling's half.

Amended 2026-09-09 on what the sweep found while closing them, because the
sentence above under-describes the memory family. "spec and memory show the read
half of the same thing" is right about spec and wrong about memory. Memory's
write paths were live and silent: `memory ingest` from a subdirectory exited 0
and laid a COMPLETE second substrate — pages, index, registry, log — beneath the
caller, and in a plain directory that is no repository at all it did the same
there; `memory ask --file-back` writes through the same core; and `memory lint`
read the store that was not there, reported 0 blockers over pages it never
opened, and wrote its run-log tree to `<caller>/.abcd/.work.local/logs/memory/`.
The spec family's write path, `spec close`, is the one that stayed a refusal
rather than a misplaced write — spec.Load returns an empty store rather than an
error, so Reconcile refused with "spec spc-N not found" for a record sitting in
the checkout, which is a true-shaped statement about the wrong store and not a
lost record. The call-site line numbers the sweep brief carried (cli.go:1918,
1958 for spec and 3386, 3426, 3482, 3518 for memory) predate the decide fix; the
sites were at 1925, 1965, 3444, 3484, 3540 and 3576 when this change started.

Two of the four halves are closed. `intent` and `ideate record` now resolve the checkout root before they read or write, through intentStoreRoot and ideateStoreRoot — the captureLedgerRoot/decideStoreRoot shape the fix direction above asked for, over the same gitutil.CheckoutRoot, with the stray-store note on the ancestor chain. All eight intent call sites (Status, CreateFromText via createIntentFromText, Plan, RecordGrounds, Ready, Link, ReEmitAudit, IngestVerdict) and the one ideate call site take the resolved root; the bare intent render stays read-only, and the two remaining refusal states exit 2 having done nothing. Per-family detectors are in internal/surface/cli/intent_root_test.go and ideate_root_test.go, including a static pass that derives the covered core functions from the core packages themselves. The record understated the ideate half in one respect: from a subdirectory the verb did not merely misaddress the store, it reported the grill's cited ids as records that "do not exist in this repository" — the operator's grill blamed for the verb's own misaddressing — and where the caller's directory carried a decision log it wrote the verdict and its pointer there and exited 0. `spec` (spec.Load and intent.Reconcile at the spec-close door) and `memory` are still open, and this record stays open for them.

## Grounds

- pursued: we expect a verb that writes or reads a record to address the checkout's store from anywhere inside it, and to refuse rather than guess outside one, so a record is never written where nobody will look and never reported absent when it is present; a verb reaching a store beneath the caller, or a plausible empty answer from a subdirectory, would show it wrong.
