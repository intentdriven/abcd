---
schema_version: 1
id: "iss-2611041115564940"
slug: "the-shell-guard-s-check-on-its-one-shot"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "the v0.16.1 cut, 2026-11-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard"
remedy: "Judge the write's target, not the line's text; a test runs the same words as an operand of git add, mv and grep."
---

The shell guard's check on its one-shot allow store refuses a harmless command when that store's directory name appears anywhere in its line: on 2026-11-04 a git add and a mv of an issue record whose generated slug happened to hold those words, and a read-only grep beside a pipe, were each refused as a write to the store.
