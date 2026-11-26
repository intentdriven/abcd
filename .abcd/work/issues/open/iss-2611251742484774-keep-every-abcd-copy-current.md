---
schema_version: 1
id: "iss-2611251742484774"
slug: "keep-every-abcd-copy-current"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "manual build of 25 November, logged as a test of what abcd relies on"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/update.go"
remedy: "Build itd-2611252040542374 once itd-2611251834407931 has shipped: on a standing yes the person gives on their own machine, abcd asks at most daily for its newest release's metadata and tells the person alone (adr-2611251808065076). The change that ships it resolves this, once whether abcd update levels the copies abcd itself provisioned is captured as an issue of its own. Installing a release unasked is iss-2611252038564722, not this. Grounds: the copies are compared from disk by the first, and only the release origin knows that a newer release exists."
related_intents: [itd-2611251834407931, itd-2611252040542374]
---

Keeping abcd current is confusing and by hand: the plugin at user scope, the plugin at project scope and the binary on PATH are updated by three separate acts, in the host and in the terminal, and an act on one leaves the others behind with nothing saying so. On 2026-11-25 a person took the update at project scope and on PATH, and their sessions went on running the user-scope copy five releases older; it took a second report to find, and a first attempt to fix it did not change that copy. The person asked for the plugin to be kept current automatically, from the terminal and from the host. adr-38 (implicit checks are disk-only) stands against abcd looking for a release on its own, so an automatic route refines or reverses it; that is for a person to confirm. Shipped beside it: the update verb for the PATH copy (itd-130) and a drive that takes each release on its yes (itd-2611062149308723). Refines iss-2610241228359748.

2026-11-25: judged as an idea, and reframed (dcn-2611251807556458). Two ambitions: every copy known, named, and compared from disk, which needs no change to adr-38; and every copy current, where a standing check and an unasked apply are two separate yeses: The check is ruled on (adr-2611251808065076, accepted 2026-11-25) and the apply is not. The verdict lists what each may never do.
