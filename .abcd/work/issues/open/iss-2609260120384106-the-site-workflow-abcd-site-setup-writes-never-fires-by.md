---
schema_version: 1
id: "iss-2609260120384106"
slug: "the-site-workflow-abcd-site-setup-writes-never-fires-by"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25 (fix round, review of lane sitesetup)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/site/setupsrc/site.yml.tmpl"
---

The site workflow abcd site setup writes never fires by itself on a repository launch scaffold laid out: auto-release runs release.yml by workflow_call so no run named release exists for workflow_run, a tag-push release run has the tag as head_branch which the branches filter drops, and a GITHUB_TOKEN-made release fires no release:published, so only a hand dispatch deploys while the verb says the site deploys on the next published release.
