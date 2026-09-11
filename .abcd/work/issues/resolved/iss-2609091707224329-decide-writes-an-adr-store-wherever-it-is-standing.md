---
schema_version: 1
id: "iss-2609091707224329"
slug: "decide-writes-an-adr-store-wherever-it-is-standing"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "release-gate"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/decide.go"
resolution: "The decide front door resolves the checkout root before it mints: decideStoreRoot asks gitutil.CheckoutRoot, which takes git's toplevel and refuses both remaining states rather than guessing — a repo-shaped tree git will not answer for, and no repository at all. A mint from a subdirectory now lands in the checkout's own decision store, and a mint outside a checkout exits 2 and writes nothing. A decision store already sitting below the checkout root is named on stderr and left untouched. The resolution is the capture verbs' own, promoted out of the ledger package: gitutil.CheckoutRoot carries the three-state policy and takes the store's noun as its only parameter, and capture.LedgerRoot is now its ledger-named door."
impact: fix
---

The decide verb hands its working directory straight to the record writer without resolving the checkout root, so it mints an architecture decision record relative to wherever the shell happens to be. Run outside any repository it exits zero, reports a repo-relative path, and creates a full decisions store in a plain directory; run from a subdirectory of a real checkout it mints into a second store beneath that subdirectory rather than into the repository's own. This is the same defect the capture verbs carried until this release, and the same remedy applies: every front door resolves the checkout root before it builds a request, and refuses rather than guessing when there is no repository to address. The evidence is not hypothetical. A checker of this release's own semantic gate, running in a scratch directory, left behind an ADR store containing one record whose slug says it was written outside any repository; nothing in the verb, its help, its surface chapter or its plugin page warned that this could happen. The cost is a decision record that no gate reads, no release cut sees and no reader finds, written with every appearance of success, and the durable record family is the one where a lost entry is least recoverable because the decision it holds was never anywhere else. Fix: resolve the root the way the capture verbs now do, refuse outside a repository with nothing written, and report a store found below the checkout root rather than adding to it. Detector: decide run outside a repository refuses with exit 2 and creates nothing, decide run from a subdirectory mints into the checkout's own store, and decide run at the root is unchanged.

## Grounds

- pursued: a decision filed outside the checkout is a decision no gate, no cut and no reader ever reaches, and the front door is the only place that knows which checkout the caller is in
