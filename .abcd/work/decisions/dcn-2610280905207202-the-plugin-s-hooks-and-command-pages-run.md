---
id: dcn-2610280905207202
slug: the-plugin-s-hooks-and-command-pages-run
date: 2026-10-28
---
2026-10-28 — The plugin's hooks and command pages run a build of the checkout's HEAD in a checkout the person lists in ~/.abcd.noindex/dogfood-roots, and the installed release everywhere else, because iss-2610070547057816 records that the person wants exactly that, so no question was put. It does not conflict with AGENTS.md's go run rule: that rule governs the agent's own invocations in a source checkout and leaves the surface half to iss-2608230943088357, which this change addresses for the hooks and command pages (facilitator on delegation, lane D, 2026-10-28).
