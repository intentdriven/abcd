---
schema_version: 1
id: "iss-2609251324599468"
slug: "the-stubs-itd-2609212130136102-leaves-for-one-release-are-to"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
deferred_after: "v0.11.1"
deferral_reason: "Deferred out loud by autonomous run A (lane drainDQ1, 2026-09-29). The removal condition is met: itd-2609212130136102 shipped in v0.11.0 with the stubs for one release, and v0.11.1 has been cut since. But removing them is a breaking change (impact: breaking), and no breaking record has reached a terminal folder since v0.11.1, so resolving this record alone moves the next release from a patch or minor bump to a breaking one. When the release that removes them is cut is a product ruling: now (the next cut is breaking), or held for the next release that already breaks something. capture defer refuses a minor record, so these two fields were set by hand (the iss-2609281654467661 precedent)."
---

The stubs itd-2609212130136102 leaves for one release are to be removed in the release after it: ahoy dry-run, ahoy identity-check, version, docs lint and site check (deprecated commands), and the bare answers of identity and ahoy remote (whose sub-verbs stay). The spec named a remainder spec minted at close for this, but a remainder close keeps the intent planned while the run closed it shipped, so the removal is held here instead; removing them narrows the surface again and needs the release to declare it.
