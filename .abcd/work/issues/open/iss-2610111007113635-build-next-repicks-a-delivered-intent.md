---
schema_version: 1
id: "iss-2610111007113635"
slug: "build-next-repicks-a-delivered-intent"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "build next lane, run-2610110908001704, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/next.go"
remedy: "Exclude from the candidates every intent a completed run in this checkout delivered (its landing merged), naming the run, or fetch the default branch and judge readiness against it; test that a planned intent whose run completed and merged is excluded with check 'run'."
---

A single 'abcd build next' can pick an intent that a completed run already delivered: the pick excludes only intents with a run still in progress, and once the landing has merged and removed the lane's branch, a checkout whose main has not been pulled still holds the intent in planned/, so the pick starts a second run for delivered work. A series excludes what it already picked, so only the single pick is exposed. Reported by the build-next implementer (run-2610110908001704).
