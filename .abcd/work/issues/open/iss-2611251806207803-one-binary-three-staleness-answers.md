---
schema_version: 1
id: "iss-2611251806207803"
slug: "one-binary-three-staleness-answers"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "manual build of 25 November, logged as a test of what abcd relies on"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/data_dir.go"
remedy: "Judge a plugin root against the version its own plugin.json names, and call the download record what it is. Grounds: the manifest is in the root being judged, and one root has one manifest whichever session asks."
---

One released binary in one plugin root gives three staleness answers, depending on which plugin data directory the environment names: on 2026-11-25 v0.17.1 said it was up to date with the plugin manifest pin (v0.17.1) with no such variable, stale against v0.16.0 with the data directory of a marketplace install, and unknown with that of an install loaded by path. The pin it names is abcd's own download record in that data directory and not the manifest of the plugin, whose version is in the root. `pluginDataDir` prefers the environment's directory, and a root provisioned under one directory is judged against another's record. Refines iss-2610241228359748.
