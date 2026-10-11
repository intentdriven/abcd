---
schema_version: 1
id: "iss-289"
slug: "the-release-workflow-builds-on-an"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "manual-capture"
found_at: "internal/core/launch/scaffold/substitutions.go"
resolution: "the toolchain pin shipped via the ci-dependency-gates branch: substitutions GoVersion and release.yml hold 1.25.13 in lockstep, TestSelfScaffoldParity green"
impact: fix
---

The release workflow builds on an unpinned Go minor: scaffold substitutions set GoVersion 1.25, which resolved to 1.25.12 on runners today — the toolchain with four stdlib vulnerabilities fixed in 1.25.13 that govulncheck flagged in ci. ci.yml was pinned to 1.25.13 the same day but release.yml renders from the template, so shipped binaries build on whatever 1.25.x the runner resolves. The v0.6.0 tag predates any fix and cannot be retro-patched (anti-tag-move); the pin lands for the next cut