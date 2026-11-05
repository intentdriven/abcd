---
schema_version: 1
id: "iss-2611050840341934"
slug: "a-major-record-captured-mid-cycle"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "coordinating build lanes, 2026-11-05"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/changelog/findings.go"
remedy: "Say what a scheduled cut does when the findings gate refuses it: which party answers, a person the repository names or a stage it trusts to fix or defer, and how the refusal reaches them, naming each record. A test refuses a scheduled cut on one major record and asserts that party's channel carries it."
---

A major record captured mid-cycle refuses the next release cut (GuardFindings), while a drain under the baseline rule takes only nitpick and minor, so on a scheduled cut with no one present one such record holds the release. The ways past it, a fix, a wontfix or abcd capture defer, are each a judgement no unattended stage is given, and nothing says who is to make it.
