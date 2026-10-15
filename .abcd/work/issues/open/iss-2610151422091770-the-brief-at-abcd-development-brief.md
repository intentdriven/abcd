---
schema_version: 1
id: "iss-2610151422091770"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.14.2 release review, iss35 brief-surface crosscheck, content commit 87513ff33"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/36-dashboard.md"
remedy: "Rewrite .abcd/development/brief/04-surfaces/36-dashboard.md:96 to match the shipped surface: `abcd dashboard stop --help` names its refusal as 'refuses when another start or stop holds the lock', and both start and stop take a file lock under ~/.abcd.noindex/dashboard/ with a 5s timeout (internal/surface/dashboard/control.go:41, :246, :642). The…"
---

The brief at .abcd/development/brief/04-surfaces/36-dashboard.md:96 says: Start refuses outside a checkout abcd manages, when a dashboard already runs, when Tailscale is not running or too old, when the tailscale command sits in the working tree or a world-writable folder, and when the port is Serve's or Funnel's; Stop (line 117) is described with no refusal at all. The shipped surface differs: `abcd dashboard stop --help` names its refusal as 'refuses when another start or stop holds the lock', and both start and stop take a file lock under ~/.abcd.noindex/dashboard/ with a 5s timeout (internal/surface/dashboard/control.go:41, :246, :642). The chapter never mentions the lock or the refusal it raises. (v0.14.2 release crosscheck, undocumented-surface, raised by opus in a27.)
