---
schema_version: 1
id: "iss-2610262031014454"
slug: "the-dependencies-page-says-abcd-needs-no"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 75 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/02-dependencies.md:3"
remedy: "In 02-dependencies.md, name the dashboard route (start refuses unless Tailscale is running and recent enough) as a second declared exception beside the runner fence, and change the opening 'no required external tools' and 'every other route still needs no external tool' to 'except the routes named here'."
finding_ref: cfk-bf8174e3f8051c97
---

The dependencies page says abcd needs no external tools beyond the planned runner fence, yet the dashboard verb refuses to start without Tailscale running

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/02-dependencies.md:3` — "abcd has no required external tools."
- End B: `.abcd/development/brief/04-surfaces/36-dashboard.md:96` — "Start refuses outside a checkout abcd manages, when a dashboard already runs on this computer, when Tailscale is not running"

The dependencies constraint says abcd has no required external tools, any present tool being an upgrade and never a prerequisite. The only exception it names is the runner route's fence. The dashboard chapter refuses start when Tailscale is not running or is too old, and the verb has no native path without it. So the dashboard route needs an external program the dependencies page says no route needs.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 75.
