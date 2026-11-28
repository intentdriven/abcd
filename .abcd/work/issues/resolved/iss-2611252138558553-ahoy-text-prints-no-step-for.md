---
schema_version: 1
id: "iss-2611252138558553"
slug: "ahoy-text-prints-no-step-for"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "review of 25 November"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/cli.go"
remedy: "Print the gap's step on the line under its title in both renders, as they do for the gaps abcd can resolve, with a test of the text form."
resolution: "Bare ahoy prints the step of a gap no install closes on the line under its title, and ahoy doctor names an advisory gap by its title with its step under it and adds the step under a diagnostic whose detail lacks it."
impact: fix
---

Bare abcd ahoy and ahoy doctor print no next step for a gap that is not required and that abcd cannot resolve itself: for a missing hooks manifest entry the step ("Update the plugin.") is printed only under --json, and ahoy doctor says nothing of it. A person sees the title and no way on. Probed on 2026-11-25 with a manifest lacking an entry.
