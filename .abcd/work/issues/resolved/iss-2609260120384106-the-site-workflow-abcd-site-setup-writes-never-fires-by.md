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
resolution: "The site workflow listens for both release and auto-release with no branch filter in the trigger, and its render gate admits the default branch or a v-tag head branch beside the fork and pull-request conditions. Impact internal: the verb is unreleased and ships with itd-2609061543533170."
impact: internal
resolved_by:
  spec: "spc-2609212141407459"
  commit: "3e23f26c"
---

The site workflow abcd site setup writes never fires by itself on a repository launch scaffold laid out: auto-release runs release.yml by workflow_call so no run named release exists for workflow_run, a tag-push release run has the tag as head_branch which the branches filter drops, and a GITHUB_TOKEN-made release fires no release:published, so only a hand dispatch deploys while the verb says the site deploys on the next published release.

## Grounds

- pursued: a release cut by the scaffolded auto-release or by a hand-pushed tag starts the site workflow with no dispatch; a first real release on a scaffolded repository that leaves the site workflow unrun would show it wrong (not yet run on Actions)
