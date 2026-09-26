---
schema_version: 1
id: "iss-2608230943260391"
slug: "abcd-version-reports-dev-and-abcd-update-calls-the-same-path"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "abcd-update-invocation-2026-08-23"
found_at: "internal/core/ahoy/update_target.go"
resolution: "The foreign update refusal names where the entry leads, what it examined and that a dev version is a build label, not the dev-shim shape; the update page says the same; abcd version already prints the install mode."
impact: fix
resolved_by:
  commit: "b6627110"
---

abcd version reports 'dev' and abcd update calls the same PATH entry 'foreign', and both are correct, which is the problem. Verified 2026-08-23 against ~/.local/bin/abcd, a symlink into a source checkout's bin/abcd-darwin-arm64. abcd version prints 'abcd dev, vintage 5d864c42bad9, up to date'. abcd update on the same entry refuses with shape foreign and the detail that the entry is not something abcd owns. Neither is wrong. The words describe unrelated properties that happen to collide. Version's 'dev' is the build-time version string, internal/core/core.go var Version = "dev", overridden by ldflags at release, so any make-build or go-run binary reports it. Update's classification is about the PATH entry's provenance: internal/core/ahoy/update_target.go resolves the occupant, isDevShimFile in store.go recognises a dev shim only by a byte prefix on the file itself, and an entry that is neither a shim, nor an owned symlink, nor a regular file falls to UpdateTargetForeign. A symlink to a hand-built binary is therefore foreign by construction and correctly so. The legibility failure is that commands/update.md lists dev-shim as its own named refusal shape with its own remedy, abcd ahoy install, so a reader who has just seen 'abcd dev' has every reason to predict that shape and gets a different one with a different remedy. The two surfaces never appear together, so nothing forces the collision into view. This is a vocabulary collision rather than a classification defect, which is why the fix is not to change the classifier. Directions, none adopted: have version name the install mode alongside the version string, so the two properties are visible in one place; or rename the build-time 'dev' to something the install-mode vocabulary does not use; or have the foreign refusal say what it examined, since a reader told an entry is not owned cannot tell whether the objection is the symlink, the target, or the absence of recorded provenance.

## Grounds

- pursued: a reader told an entry is foreign can see what was examined and why 'dev' does not mean the dev-shim shape; shown wrong if TestPlanForeignRefusalNamesWhatItExamined finds the detail silent on either
