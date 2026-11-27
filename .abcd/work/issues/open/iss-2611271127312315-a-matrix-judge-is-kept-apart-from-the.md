---
schema_version: 1
id: "iss-2611271127312315"
slug: "a-matrix-judge-is-kept-apart-from-the"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
remedy: "Have judgeApart compare the judge's model with every builder's served model, launched runner cells included, and refuse or mark the pairing when they match; add a test with a launched runner builder whose model equals the host judge's."
---

A matrix judge is kept apart from the builders it judges by comparing models (judgeApart, internal/core/lab/matrix/checkflow.go:189), but it never compares a host judge with the model a launched runner cell's builder was served. Now that runner cells launch, a host judge can score a copy built by its own model.
