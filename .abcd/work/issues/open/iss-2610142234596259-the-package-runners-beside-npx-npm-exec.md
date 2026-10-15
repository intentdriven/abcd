---
schema_version: 1
id: "iss-2610142234596259"
slug: "the-package-runners-beside-npx-npm-exec"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "security review of feat/guard-port-blocks for v0.15.0"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/match.go"
remedy: "Read the sibling runners as wrappers: add pnpm exec, yarn exec and bun x to runnerSubcommands, bunx and pnpx beside npx, and yarn's bare <pkg> form where the word is not a yarn subcommand, each with its documented value flags; prove it with kill-port and pkill behind each runner at block, and their everyday forms (pnpm exec eslint, bunx prettier) at allow."
---

The package runners beside npx, npm exec, pnpm dlx and yarn dlx are not read as wrappers, so a blocker behind them only warns as an unrecognised launcher: pnpm exec, yarn <pkg>, yarn exec, bunx, bun x and pnpx each run the program their first operand names, and pnpm exec kill-port 3000 or bunx kill-port 3000 warns where npx kill-port 3000 blocks.
