---
schema_version: 1
id: "iss-2611040555110508"
slug: "06-delivery-02-verification-matrix-md-48"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/02-verification-matrix.md"
remedy: "Say the include set comes from launch-payload.json, the .abcd/ deny is structural, and no file overrides that deny."
---

06-delivery/02-verification-matrix.md:48 says include and exclude are resolved from the bundler's own rules and, as staged, that the launch core reads no per-repo allow-file. internal/core/launch/includes.go:14 reads .abcd/config/launch-payload.json, and 04-surfaces/04-launch.md:508-510 calls that list the only mechanism that can put a path into the release artefact. Only the structural deny of .abcd/ is the bundler's own, and the file cannot override that.
