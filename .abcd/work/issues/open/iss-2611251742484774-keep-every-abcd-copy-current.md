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
remedy: "Two parts. Without touching adr-38: let `abcd update` bring every installed copy it finds to one release in one act, plugin scopes included, calling the host to update its own, and have a session start compare the copies on disk and name the one behind. Needing a ruling on adr-38: an opt-in, off until the person turns it on, by which the host or abcd takes each new stable release unasked. Grounds: the copies are all readable on disk, and the drive already takes releases under a standing yes."
---

Keeping abcd current is confusing and by hand: the plugin at user scope, the plugin at project scope and the binary on PATH are updated by three separate acts, in the host and in the terminal, and an act on one leaves the others behind with nothing saying so. On 2026-11-25 a person took the update at project scope and on PATH, and their sessions went on running the user-scope copy five releases older; it took a second report to find, and a first attempt to fix it did not change that copy. The person asked for the plugin to be kept current automatically, from the terminal and from the host. adr-38 (implicit checks are disk-only) stands against abcd looking for a release on its own, so an automatic route refines or reverses it; that is for a person to confirm. Shipped beside it: the update verb for the PATH copy (itd-130) and a drive that takes each release on its yes (itd-2611062149308723). Refines iss-2610241228359748.
