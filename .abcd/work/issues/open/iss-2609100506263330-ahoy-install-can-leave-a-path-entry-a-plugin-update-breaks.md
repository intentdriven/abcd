---
schema_version: 1
id: "iss-2609100506263330"
slug: "ahoy-install-can-leave-a-path-entry-a-plugin-update-breaks"
severity: "major"
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

---

**Evidence (2026-09-23, folded in from iss-2609120447482506, closed as a
duplicate of this record):** the same install shape observed on a second
machine during an `ahoy install` onboarding on 2026-09-12. The PATH entry was
written as a symlink to the plugin-root binary because the cache held no
verified artefact; immediately after a plugin update the link still pointed at
the superseded root while the live root had moved on, `install_mode` read
`pinned`, and bare `ahoy` reported zero gaps. Two asks it made, restated here so
they survive the close:

- **Surface the pin as a gap while it still works**, not only in the one-time
  install note: an entry that is a symlink into a plugin root rather than an
  owned copy is the state this record describes. The superseded half of that
  ask is answered on main by 177a407d (iss-2609161805447092): a pin into an
  older vintage now raises the required, resolvable `symlink.superseded` gap.
  A pin into the CURRENT root on a cold cache still raises nothing; the
  `symlink.legacy` gap fires only once a verified cache is available.
- **Provision the cache during install**, so the owned copy is written first
  time — this record's first "Needed" option.

**Severity raised to major (2026-09-23).** This record was filed minor;
iss-2609120447482506, closed as its duplicate, was major. The close folded that
record's unanswered half in here (a pin into the current root on a cold cache
raises no gap, and install does not provision the cache), so its severity comes
with it: closing a major as a duplicate of a minor would drop the finding below
the release-cut guard without anyone ruling it minor.
The release-cut guard still does not see it: the guard reads the records that
entered `open/` since the anchor, and this record sat in `open/` at v0.9.0, so
it counts as standing backlog, while iss-2609120447482506 entered after that
anchor. The raised severity keeps the grade honest; it does not put the finding
back in front of this cut's guard.
