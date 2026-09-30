---
schema_version: 1
id: "iss-2609302205483845"
slug: "namescodeowners-internal-core-implement-loop-land-go-is"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
remedy: "Read CODEOWNERS the way the forge does: the first file found in .github/, the root, then docs/ is the only one read (GitHub docs, 'About code owners': the first CODEOWNERS file found in those locations is used), and a line counts only when an owner token follows its pattern: @username, @org/team or an e-mail address; a pattern-only line, a bare '@' and anything after a '#' name nobody."
---

namesCodeOwners (internal/core/implement/loop/land.go) is looser than the forge's CODEOWNERS parse: it counts any non-comment line as naming an owner and ORs all three CODEOWNERS locations, so a code-owner review arms auto-merge where the forge asks nobody: a line '* @' (an owner-less token), a pattern-only line, and a comment-only .github/CODEOWNERS shadowing a root CODEOWNERS that names an owner each arm (review-amApproval, minor).
