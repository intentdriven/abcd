---
schema_version: 1
id: "iss-2610262032429930"
slug: "configuration-lists-every-abcd-config"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 306 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md:7"
remedy: "Edit 03-configuration.md (opening paragraph and the config.json schema): add the repository-layer `release.digest` key (read by internal/core/release/digest.go) to the opt-in keys the binary reads."
finding_ref: cfk-3fe2a75d9e49c394
---

configuration lists every .abcd/config.json key the binary reads, without release.digest; launch's cut reads release.digest from that file

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/03-configuration.md:7` — "Beyond those sixteen, the binary reads only keys a person opts into"
- End B: `.abcd/development/brief/04-surfaces/04-launch.md:118` — "`release.digest`, a repository-relative path to a `.json` file"

The configuration chapter names the sixteen keys and then says that beyond them the binary reads only an enumerated set of opt-in keys. release.digest is in neither list. The launch chapter says the cut reads release.digest from the repository's own .abcd/config.json and appends the release to that file, and that abcd itself declares one. The binary therefore reads a config.json key the configuration chapter says it does not.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 306.
