---
schema_version: 1
id: "iss-2610120955431790"
slug: "abcd-ahoy-install-adopts-an-existing"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "security review of the provenance tag binding on fix/sec-installers-261012"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/apply.go"
remedy: "On the cold-cache adoption path, hash the owned copy and run the same tag-bound provenance check before reporting it adopted (or report it as adopted unverified, naming the hand check), so the claim holds for every entry install accepts."
---

abcd ahoy install adopts an existing owned PATH copy on a cold cache without verifying its build provenance: when the plugin cache is missing or unbound and the PATH entry is already an owned copy, install returns and keeps it as it stands (internal/core/ahoy/apply.go, the binTargetOwnedCopy return in the cold-cache branch). It writes nothing, so it is not an unverified write, but it is a gap in the claim that every promotion is verified: the copy it adopts was never checked by this run against the provenance release.yml signed, nor against the release commit.
