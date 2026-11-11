---
schema_version: 1
id: "iss-2611101320265526"
slug: "abcd-capture-files-one-record-per-call"
severity: "minor"
category: "ux"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/workflow.go"
remedy: "Add a batch form of capture that reads several records from one file, checks them all before writing any, and writes them under one lock; a test files three records from a file and finds all three, and a file with one bad record writes none."
---

abcd capture files one record per call, and the guard refuses a helper script that writes and runs calls to the record verb, so filing many records takes many calls. On 7 Nov filing 31 records took 31 calls, one per record.
