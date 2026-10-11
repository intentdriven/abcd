---
schema_version: 1
id: "iss-2608241845109280"
slug: "abcd-site-build-refuses-a-committed"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "v0.6.4 release PR 2026-08-24"
found_at: ".abcd/work/issues/resolved/iss-2608241347321757-resolving-an-issue-is-a-convention.md"
resolution: "the record's indented code block is fenced, and the class is gated three ways: a make site-render target that builds into a throwaway directory, preflight depending on it so the pre-push hook catches it, and a Site-render gate step on ci.yml's always-run Linux lane so a records-only pull request is covered. site-screenshots.yml's path filter is unchanged but its comment now says what it actually audits — layout — instead of claiming to watch everything that can change the rendered page."
impact: fix
---

abcd site build refuses a committed record, so the site render fails on main and nothing catches it before CI. .abcd/work/issues/resolved/iss-2608241347321757-...md:52 carries a four-space indented code block, and the renderer supports a fixed subset only: 'unsupported markdown construct: indented code block: fence code with backticks so it renders as a copyable command — the site renders a fixed subset and never passes text through unrendered'. The binary is behaving as designed; the record is non-conforming. It reached main because the site-screenshots workflow is path-filtered and the ci changes classifier stands jobs down for a pull request confined to .abcd/, so PR 489 never ran a site build, and the failure only surfaced on the first later PR touching a watched path (the v0.6.4 release PR 497). Two gaps, not one: the record's markdown is wrong, and record-lint does not check records against the renderer's supported subset even though every record under .abcd/work/issues/ is site-rendered. The second is the detector; the first is its acceptance corpus. Note screenshots is not a required status check, so this never blocked a merge.