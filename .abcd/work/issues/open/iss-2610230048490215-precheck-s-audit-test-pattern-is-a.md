---
schema_version: 1
id: "iss-2610230048490215"
slug: "precheck-s-audit-test-pattern-is-a"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "review pair of chore/precheck-and-records, 2026-10-23"
origin: researcher-authored
production_mode: hand-written
found_at: "Makefile"
remedy: "Mark the audit tests explicitly (a shared helper that reads a committed file, or a build tag) and have precheck select by that mark, with a test that every test reading a committed prose or generated file through the helper is selected"
---

make precheck chooses the command and site packages' audit tests by a word in their names (PRECHECK_AUDIT_TESTS), and nothing checks that a new test reading the committed tree carries one: an audit test named outside the pattern runs in preflight but not in precheck, so a lane can pass precheck and still lose a preflight slot to it. A structural detector was tried and set aside: about 250 command-package tests reach the repository root, most to run scripts and hooks rather than to audit prose, so 'reads the root' does not tell an audit from a behaviour test.
