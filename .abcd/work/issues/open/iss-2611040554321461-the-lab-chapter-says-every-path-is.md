---
schema_version: 1
id: "iss-2611040554321461"
slug: "the-lab-chapter-says-every-path-is"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/lab_matrix.go"
remedy: "Pass the declaration path through fsutil.RedactHome where the path is displayed, keeping the shown command runnable, or narrow the chapter's sentence."
---

The lab chapter says every path is written in tilde form (.abcd/development/brief/04-surfaces/31-lab.md:180). The matrix plan's 'to start it' line and its JSON approve_with embed the declaration argument unredacted (internal/surface/cli/lab_matrix.go:90), so a declaration under the home directory given by absolute path is printed with that absolute path. The refusal paths of the same verb do redact it.
