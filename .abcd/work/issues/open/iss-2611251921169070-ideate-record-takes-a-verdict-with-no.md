---
schema_version: 1
id: "iss-2611251921169070"
slug: "ideate-record-takes-a-verdict-with-no"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "manual build of 25 November, logged as a test of what abcd relies on"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/ideate.md"
remedy: "Tell the two apart in the payload (opened and unproven, or not opened with the reason), and refuse a survives verdict whose load-bearing claims were all never opened."
---

abcd ideate record takes a verdict of survives whose only research claim is marked unverifiable against a web address nobody opened: the dry run passes. The page calls an honest unverifiable a result, and the record cannot tell a source opened and found to prove nothing from one the session had no tool to open, so a verdict can stand on a missing tool. Tested on 2026-11-25 with the dry run.
