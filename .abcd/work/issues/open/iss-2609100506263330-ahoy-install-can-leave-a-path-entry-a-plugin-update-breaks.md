---
schema_version: 1
id: "iss-2609100506263330"
slug: "ahoy-install-can-leave-a-path-entry-a-plugin-update-breaks"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous-run field experiment in a managed repository, 2026-09-07/08; re-filed into abcd 2026-09-10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal (ahoy install, PATH entry)"
---

`ahoy install` can report success while leaving a PATH entry that a later plugin update silently breaks, and the condition that decides which happens is invisible to the operator.

Observed adopting abcd in a managed repository. The install reported `clean` and `install: pinned`, and `abcd` worked. It also emitted a note: no verified release artefact was present in the persistent plugin data directory, so the entry was written as a SYMLINK into the versioned plugin cache directory rather than as an owned copy, and that directory is replaced when the plugin updates, at which point the entry dangles. The remedy given is to "re-run `abcd ahoy install` from a session whose hooks have provisioned the cache", which is a condition the operator has no way to check, create, or even observe from the outside.

The note is loud, which is right, and the failure is exactly the one this repository had already been bitten by from a different cause — a dangling user-bin entry shadowing everything, filed separately. So the tool knows this state is bad, warns about it, and installs into it anyway.

Needed: prefer fetching and verifying the release artefact at install time when the cache is cold, rather than degrading to a link that is known to break; or, if the fetch is deliberately out of scope for `install`, refuse the link form and say what to run first, since a warned-about install that dangles later is harder to diagnose than a refusal now. Either way, `ahoy` should notice on a later run that its own entry has become dangling and offer to repair it, which it currently cannot do because a dangling entry classifies as foreign.
