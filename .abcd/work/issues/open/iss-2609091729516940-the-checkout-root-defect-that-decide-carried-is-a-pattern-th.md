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
---

The checkout-root defect that decide carried is a pattern the surface shares, not two instances of it. iss-2609090951291524 fixed it for the seven capture verbs and iss-2609091707224329 for decide; the same raw shape — os.Getwd() handed to a core that joins a repo-relative store path onto whatever it is given — remains at the intent, spec, ideate and memory front doors, which between them own the intent store, the spec store, the ideate verdict record and the memory substrate. Reproduced against a scratch git checkout holding one intent draft: abcd intent run from internal/core reports drafts 0, planned 0, shipped 0 while the identical binary at the root reports the draft, and abcd intent "<text>" run from the same directory mints a second store at internal/core/.abcd/development/intents/drafts and reports a repo-relative path that reads exactly like the checkout store's. Run in a plain directory that is not a repository at all it exits 0 and lays the full intent skeleton there. spec and memory show the read half of the same thing from a subdirectory. The cost is the one the two closed records already argue: a record filed where no gate, no release cut and no reader looks, written with every appearance of success, and for the intent store an intent whose spec can never be closed against it. Fix direction: the resolution is already shared and already generalised — gitutil.CheckoutRoot takes git's toplevel, refuses a repo-shaped tree git will not answer for, and refuses when nothing repo-shaped sits above, with the store's noun as its only parameter — so each remaining front door needs a resolving step of the captureLedgerRoot and decideStoreRoot shape and the strayStoreNotes call that reports what an unresolved door already deposited, not a new resolver. Detector, per family: the verb run from a subdirectory addresses the checkout's store, the verb run outside a repository refuses with exit 2 and writes nothing, the verb run at the root is unchanged, and a static pass over the surface package refuses a request built from an unresolved working directory.
