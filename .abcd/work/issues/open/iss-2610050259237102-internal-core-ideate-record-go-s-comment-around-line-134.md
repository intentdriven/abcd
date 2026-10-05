---
schema_version: 1
id: "iss-2610050259237102"
slug: "internal-core-ideate-record-go-s-comment-around-line-134"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "v0.13.0 release cross-check, 2026-10-05"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ideate/record.go"
remedy: "Test the record writer with an in-root relative symlink as an intermediate component; correct the comment to what os.Root guarantees (escape refusal, not every symlink), and if the writer relies on refusing every symlink, add an explicit Lstat check per component."
---

internal/core/ideate/record.go's comment (around line 134) asserts that os.Root refuses every symlinked path component, but os.Root follows a relative symlink that stays inside the root; the release cross-check (x-044, v0.13.0) raised it and could not reproduce a write through it, so the claim and whatever guarantee leans on it are unverified.
