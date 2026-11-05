---
schema_version: 1
id: "iss-2611050819312152"
slug: "a-walled-stage-s-shell-finds-no-cache-of"
severity: "minor"
category: "tech-debt"
source: "review-followup"
found_during: "the drive's security review, 2026-11-05"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/runner/wall.go"
remedy: "Let the person's machine configuration name further cache variables a stage carries (each an absolute path outside the login), read by stageGoExports' successor, with the shipped default keeping only Go's."
---

A walled stage's shell finds no cache of a toolchain other than Go's: its home is emptied at each launch and only Go's caches are named, so a repository built with rustup, cargo, npm or the like downloads or fails in every stage, though abcd serves any repository's product.
