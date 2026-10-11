---
schema_version: 1
id: "iss-350"
slug: "readme-promises-abcd-ahoy-reports-the"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "bughunt-round-2"
found_at: "README.md"
resolution: "README promises the PATH hint from ahoy install's reachability note, the verb that prints it"
impact: fix
---

README promises abcd ahoy reports the local-bin-not-on-PATH state as a named gap after the copy one-liner install, but detectBinDirOnPath is suppressed for a copy install: no plugin root yields no PATH gap at all and a resolvable root yields symlink.foreign with resolve-manually
## Evidence

- `README.md:206-208` sits inside the CLI copy-install paragraph and contrasts itself with the installer in its own clause, so it promises the bare read-only verb. Reproduced with scratch HOMEs: after the copy one-liner with `~/.local/bin` off PATH, no plugin root yields no PATH gap at all (`detectPathSymlink` returns nil at `internal/core/ahoy/detect.go:390-392`); a resolvable root yields `symlink.foreign` ("Resolve manually; ahoy refuses to clobber") because `detectBinDirOnPath` is suppressed unless `installed` (`detect.go:484-493`). The promised `path.bin_dir_not_on_path` gap fires only for an abcd-owned symlink install.
- `abcd ahoy install` does print the export line (`noteReachability`, `apply.go:854-873`) — but that is the verb README names separately, and it costs an adopting write.
- Refuter verdict: CONFIRMED substantive (minor, documentation): false for 100% of readers who follow the CLI section and land off PATH. Doc-side qualification is the right minimal fix; the engine question (PATH state gated on plugin-root resolvability) is a separate design call.
