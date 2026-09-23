---
schema_version: 1
id: "iss-2608241347321759"
slug: "isnull-misses-trailing-comment-null-form"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "pr-294"
found_at: "internal/core/frontmatter/frontmatter.go"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (ruling owed: the frontmatter line scanner trims unquoted trailing comments (NULL # x reads null) or documents the form unsupported). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

frontmatter.Fields trims the captured value but strips nothing after a comment marker, and IsNull compares exact strings, so superseded_by: NULL # no successor is read as the literal string 'NULL # no successor' rather than the null token -- a trailing-comment form valid YAML never resolves as null; follow-up to pr-294 per reviewer (frontmatter self-describes as a line scanner, so decide: trim unquoted trailing comments or document as unsupported) takeover 2026-08-24