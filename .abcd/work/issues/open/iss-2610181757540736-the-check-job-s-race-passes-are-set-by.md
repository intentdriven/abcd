---
schema_version: 1
id: "iss-2610181757540736"
slug: "the-check-job-s-race-passes-are-set-by"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop"
remedy: "Measure git processes per package across the whole loop, cli and claim suites, not one test, find the call sites that dominate there, and cut those, each change judged by the package-wide count and then by merge-group race-pass time over several runs"
---

The check job's race passes are set by their total work: the macOS pass keeps a steady 2.80 to 2.85 of its runner's three test slots busy and the ubuntu one about 3.4 of four, so no reordering shortens them, and their slowest packages (internal/core/implement/loop, internal/surface/cli, internal/core/claim, about 46% of the macOS pass) spend most of it in git processes. A first cut, a default-branch name per claim operation and a batched view (#1099, #1100), took one drain test from 1,107 git processes to 830 but the whole of the loop and claim packages only from 126,334 to 119,327 (5.5%), which CI could not tell from the difference between runners; a cut measured on one test overstates the package.

On 2026-11-01 the macOS pass ran into the job's cap in a merge group and dropped two pull requests (#1326, #1322) with no test failing. The watchdog stopped it at its 1h20m deadline (the 85-minute cap less its 5-minute margin), run 41317526495. In that run internal/surface/cli took 1203s, internal/core/implement/loop 1511s and internal/adapter/scanner 1133s under the race detector, and internal/core/memory, internal/core/release and internal/core/repolint were still running when it was stopped. Since the pass already keeps its three slots busy, splitting it across separate jobs is the only change that shortens one job, besides cutting the work this record measures. Both pull requests merged after one re-arm.
