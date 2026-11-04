---
schema_version: 1
id: "iss-2611040554318883"
slug: "the-lab-chapter-says-the-matrix-plan-is"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/31-lab.md"
remedy: "Name the passing preflight as a precondition in the chapter's matrix section and in the verb's refusal list."
---

The lab chapter says the matrix plan is built from the declaration and the machine's configuration alone (.abcd/development/brief/04-surfaces/31-lab.md:34), and `abcd lab matrix --help` enumerates its exit 2 refusals. The verb also refuses, exit 2, a lab with no passing preflight (internal/core/lab/procedure.go:58-60, reached from internal/core/matrix/consent.go:126). Neither text names that refusal.
