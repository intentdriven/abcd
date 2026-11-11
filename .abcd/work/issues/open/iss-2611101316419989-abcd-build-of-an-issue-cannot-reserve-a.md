---
schema_version: 1
id: "iss-2611101316419989"
slug: "abcd-build-of-an-issue-cannot-reserve-a"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/foundat.go"
remedy: "Take a single token with no slash and no extension as a path when that file exists at the root of the checkout (Makefile, LICENSE, Dockerfile), at capture and at build; a test builds an issue whose found_at is Makefile and finds it reserved."
---

abcd build of an issue cannot reserve a found_at that names a file at the repository root with no extension. foundAtPath (internal/core/capture/foundat.go) takes a value as a path only when it holds a slash or ends in a file extension, so Makefile reads as a conceptual location and build refuses the issue as having no found_at path. Seen on 7 Nov for an issue whose found_at was Makefile; the rule is unchanged at this branch's head.
