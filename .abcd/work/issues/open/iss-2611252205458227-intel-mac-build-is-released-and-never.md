---
schema_version: 1
id: "iss-2611252205458227"
slug: "intel-mac-build-is-released-and-never"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "review of 25 November"
origin: researcher-authored
production_mode: hand-written
found_at: "Makefile"
remedy: "Run the Intel build on a hosted Intel macOS runner (macos-26-intel or macos-15-intel, free for a public repository) nightly and at each release: the smoke lane first, the whole suite if its time allows; or stop releasing that build and say so in the README."
---

abcd releases a build for Intel Macs (darwin/amd64) and its README names macOS on amd64 as a released platform, but nothing ever runs that build: every macOS job runs on Apple silicon, the smoke lane builds for the machine it runs on, and a developer Mac on macOS 27 cannot run an Intel program ("Bad CPU type in executable", tried on 2026-11-25). A fault only that build has is first met by a person who installed it. iss-325 closed the compile half; this is the running half.
