---
schema_version: 1
id: "iss-2611211344505614"
slug: "abcd-lab-matrix-and-abcd-lab-matrix"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "build run-2611202324382832 lane-5 (spc-2610161949559050 step 8)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/lab_matrix.go"
remedy: "add a matrix Starter that drives the loop's trial mode per cell, makes each copy with makeCopyClone, scores it on its hidden tests under checkCompliance, writes it with writeTrial and records reviews with recordReviews, and use it in place of recordOnly"
---

abcd lab matrix and abcd lab matrix rerun record a consented run and launch nothing: the front door's Starter is recordOnly, and nothing joins the loop's trial mode to the matrix's clones, hidden tests, compliance check, reviewers, judging and trial writer, so no finished run exists for the report to read end to end
