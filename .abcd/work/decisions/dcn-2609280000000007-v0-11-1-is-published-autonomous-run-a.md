---
id: dcn-2609280000000007
slug: v0-11-1-is-published-autonomous-run-a
date: 2026-09-28
---
2026-09-28 — v0.11.1 is published: autonomous run A approved the `release` environment at 22:44:40Z under ruling A2 and the releases ruling of 2026-09-25T08:04:52Z, after the merge queue, the verify job, the tag job and main's own CI on the tagged commit 2bc519f7 reported green (every check run succeeded apart from those skipped by design; the macOS leg of the push CI was the last to report). The release published at 22:47:00Z with four binaries, the plugin archive, checksums.txt and the site archive; the run verified the darwin-arm64 binary and the plugin archive against checksums.txt, `abcd --version` reports v0.11.1, the plugin archive's SHA-256 equals the digest the catalog pins and its address answers, and the build-provenance attestation verifies as signed by release.yml on main. Unlike v0.11.0, the site rendered and deployed inside the release run, so no redeploy was needed; abcdev.app shows v0.11.1. The version is v0.11.1 rather than the v0.12.0 the run had expected, because the cut derives the version from the records and nothing since v0.11.0 is breaking.
