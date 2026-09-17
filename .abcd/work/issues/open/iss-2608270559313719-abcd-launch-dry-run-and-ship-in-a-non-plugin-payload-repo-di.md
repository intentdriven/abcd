---
schema_version: 1
id: "iss-2608270559313719"
slug: "abcd-launch-dry-run-and-ship-in-a-non-plugin-payload-repo-di"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "testimony-launch-dryrun-2026-08-27"
found_at: "internal/core/launch/includes.go"
---

abcd launch --dry-run (and ship) in a NON-plugin-payload repo dies with a raw 'include config not found: .abcd/config/launch-payload.json' (LoadIncludes preflight in internal/core/launch/includes.go), giving the operator no idea WHY. launch preview/ship is a plugin-payload-repo feature (it needs launch-payload.json, and ship additionally needs .claude-plugin/plugin.json); a repo that ships no plugin bundle legitimately has neither and should be told so, not handed a missing-file error. Fix (loud-staging/legibility): when the launch config is absent AND the repo is not a plugin-payload repo, the dry-run should explain 'launch preview/ship applies to plugin-payload repos; this repo's release path is launch scaffold + the CHANGELOG roll + auto-release' rather than reporting a raw missing include config. Surfaced from a Testimony (non-plugin repo) onboarding session.

**Corroboration (2026-09-10, autonomous-run field experiment in managed
repositories).** Reproduced twice more, and in one case it blocked the task
outright rather than merely confusing the operator.

First reproduction: an operator ran the bare verb and was sent down a two-step
dead end. `abcd launch` answered "pass --dry-run to preview the bundle
(publishing is not wired at this stage)", advertising a flag without checking
whether this repo can run it; `abcd launch --dry-run` then answered "include
config not found: .abcd/config/launch-payload.json" and stopped. Neither message
says the thing the operator needs: this repository does not use `abcd launch`,
its releases are cut by a changelog-driven workflow, and the correct move is to
roll a dated heading and merge. The repo is not misconfigured — `abcd launch
scaffold` exists precisely to install that changelog-driven gate, and this repo
has one and it works — so the tool routed an operator away from the mechanism
its own sibling verb installed.

Second reproduction: a session could not run the release preview at all, on a
repository that had already cut releases through abcd (0.3.1 and 0.4.0 headings
stand in its CHANGELOG). Nothing in `--help`, `abcd lint` or `abcd ahoy` names
the missing include file or says how to create one, and `launch scaffold`
documents only the workflows and the runbook. The preview was abandoned.

Two remedies beyond the one already recorded here. Have the bare invocation
detect a changelog-driven gate and name it, rather than advertising a flag that
cannot run. And consider whether the dry run needs a payload include at all —
the gates and the changelog composer do not — in which case the preview should
run without one.
