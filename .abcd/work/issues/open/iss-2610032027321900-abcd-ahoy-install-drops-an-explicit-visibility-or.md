---
schema_version: 1
id: "iss-2610032027321900"
slug: "abcd-ahoy-install-drops-an-explicit-visibility-or"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "itd-2610030814013772 step 2 re-verify 2026-10-03"
origin: researcher-authored
production_mode: hand-written
remedy: "On the declined return in stepConfigValues, add one note per explicit value override whose value differs from the saved one (--visibility, --docs-target, --oracle-backend, --scan-deep), naming the flag and its value, saying it was not applied because the settings change was declined so .abcd/config.json was not written, and how to apply it (run abcd ahoy install again and answer y to the config-change question). The malformed-config path already names dropped overrides the same way (droppedOverrides), which is the precedent."
---

abcd ahoy install drops an explicit --visibility or --docs-target flag without a word when the person declines the config-change category while a required setting is still missing: stepConfigValues returns before the settings write, the deferred rollback takes the echoed change back (so the receipt no longer claims it), but no note says the flag the person typed was not applied, so the run reads as if the flag were honoured or ignored by design.
