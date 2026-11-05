---
schema_version: 1
id: "iss-2611051719379115"
slug: "a-pull-request-body-naming-a-record"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "build run, 5 November"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/landing/land.go"
remedy: "`abcd land` reads an existing pull request's body, runs the resolution check against it, and says why when it arms or finds a pull request whose body fails."
---

`abcd land` runs the mention check only on a body it composes, and never reads the body of a pull request that already exists (`PRStateQuery`, `internal/core/landing/land.go`). So a body written by hand that names a record without a `Refs:` or `Resolves:` line fails the resolution gate within about a minute of the pull request opening, and a later land run names the `attribution` check as held without saying why. This happened on #1408 on 5 November.
