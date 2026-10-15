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

ahoy install --outbound-hook leaves git config --local abcd.binary for each person to set by hand in each clone, so the commit-message check warns and lets every commit through until someone does, although the recorded remedy of iss-2609250834251447 said ahoy install would write it to the owned PATH copy, the abcd.guardPath precedent. The product thinker ruled on 2026-10-15 that the installer writes abcd.binary itself, to the absolute path of the provenance-verified abcd copy ahoy installs.

The hooks decision, made in the fix round of 2026-10-15: the product thinker ruled that the commit-message check runs with no manual step, within the rule that abcd changes nothing about how a repository works beyond what the person opted into. On the explicit --outbound alone, where no core.hooksPath is set at any scope and the common git directory's hooks/commit-msg does not exist (resolved with git rev-parse --git-common-dir, so every worktree shares it), install writes abcd's embedded commit-msg template there, marked as abcd's. It finds abcd through abcd.binary as the committed copy does, and never runs or calls the committed .githooks copy, so no repository-authored code runs through it. It never replaces an existing hook, never sets core.hooksPath and never touches global config: where a hooks path is set, or a commit-msg hook abcd did not write is there, it writes nothing and one note says the check is not wired, why, and the line to chain it with. A rerun leaves a current abcd-marked copy as it is and refreshes a stale one. The binary abcd.binary names is not re-verified at commit time; a swap of the PATH copy by the same account is out of scope.
