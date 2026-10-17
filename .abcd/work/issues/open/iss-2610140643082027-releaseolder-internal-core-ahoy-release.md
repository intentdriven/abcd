---
schema_version: 1
id: "iss-2610140643082027"
slug: "releaseolder-internal-core-ahoy-release"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "security review of ddf25214d (cold-cache newest-release mark)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/release_mark.go"
remedy: "When the cores are equal and both tags carry a pre-release, order the pre-release identifiers by SemVer 2.0 §11 (numeric numerically, alphanumeric lexically, fewer fields lower) in one exported comparator beside launch.CoreGreater, and test that v0.14.0-rc.1 is older than v0.14.0-rc.2 and v0.14.0-rc.10 newer than v0.14.0-rc.9."
---

ReleaseOlder (internal/core/ahoy/release_mark.go:50-62) compares cores only and then asks whether a alone carries a pre-release, so two pre-releases of one version never compare older: ReleaseOlder("v0.14.0-rc.1", "v0.14.0-rc.2") is false, confirmed at tip. A machine whose newest-release mark is v0.14.0-rc.2 therefore takes a genuine v0.14.0-rc.1 build at every place the mark is the floor: the cold-cache PATH copy check (apply.go checkOwnedCopy), the warm cache promotion (owned_copy.go:467) and abcd update's withdrawal check (internal/surface/cli/update.go:102). A pre-release rollback inside one version slips the floor the mark exists to keep. Pre-existing; low reach while abcd publishes no pre-release tags.
