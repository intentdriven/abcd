---
schema_version: 1
id: "iss-2610152024267547"
slug: "ahoy-install-outbound-hook-leaves-git"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "product thinker ruling on the outbound opt-ins, 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/install_summary.go"
remedy: "On the explicit outbound opt-in only, write git config --local abcd.binary to the absolute path of the owned PATH copy whose build provenance this run verified, and refuse, writing nothing and saying so, a copy whose provenance was not verified, the dev shim, a relative path or a copy inside the working tree; prove each with a test watched failing first."
resolution: "ahoy install --outbound writes this clone's git config --local abcd.binary itself, to the absolute path of the PATH copy whose build provenance the same run verified, so the commit-message check runs without a hand step. It writes nothing and says why for a copy whose provenance did not verify, the dev shim, a relative path or a copy inside the working tree, and leaves a value the person already set (without printing it). Only the explicit opt-in writes it, it is clone-local and never committed, and install still leaves core.hooksPath to the person, as the name guard and the attribution prompt do"
impact: additive
resolved_by:
  commit: "06333892658848ab003a8323ac143e8d07d32052"
---

ahoy install --outbound-hook leaves git config --local abcd.binary for each person to set by hand in each clone, so the commit-message check warns and lets every commit through until someone does, although the recorded remedy of iss-2609250834251447 said ahoy install would write it to the owned PATH copy, the abcd.guardPath precedent. The product thinker's answer, 2026-10-15 ~20:15 BST: "Installer sets it up": ahoy install --outbound-hook writes abcd.binary itself, to the absolute path of the provenance-verified abcd copy ahoy installs.
