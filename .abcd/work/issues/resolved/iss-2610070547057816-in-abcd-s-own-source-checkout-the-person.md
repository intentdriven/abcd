---
schema_version: 1
id: "iss-2610070547057816"
slug: "in-abcd-s-own-source-checkout-the-person"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "a /abcd:capture in abcd's own checkout, 2026-10-07"
origin: researcher-authored
production_mode: hand-written
found_at: "hooks/hooks.json"
remedy: "A per-machine opt-in list, ~/.abcd.noindex/dogfood-roots (one absolute checkout path per line, owned by the account and writable by no one else, the trusted-roots shape): inside a listed checkout the hook shims and the command pages' resolution ladder run a binary built from that checkout's HEAD, rebuilt when HEAD moves, and everywhere else the installed stable binary, unchanged; a test proves an unlisted checkout, a posing module path included, never reaches the build rung."
resolution: "A checkout the person lists in ~/.abcd.noindex/dogfood-roots, each line naming the checkout, a tab, the URL of the remote its reviewed branch lives on, and optionally a tab and the branch (main when left out), read through fsutil.HomeDeclarationEntry with the trusted-roots checks, runs in every hook shim and through the command pages' ladder a build of that branch's tip as the declared remote reports it. A background builder reads the tip with git ls-remote (isolated environment, no credential prompt, bounded in time and output), records it in the store under the build lock at 0600, and builds it once the checkout holds it, the commit and its trees re-hashed with replacement objects off and the export checked blob by blob; the hook never touches the network and names a build only for that recorded tip, so the checkout's HEAD, refs, origin/HEAD and remote config choose nothing and a moved ref only prompts one more check. A line with no remote, an unreadable remote, an unfetched tip or a failed build leaves the installed release running, said on stderr; everywhere else the installed release runs unchanged, and an unlisted checkout posing as abcd never reaches the build. Anything running as the account can still edit the declaration or replace a file in the store."
impact: additive
---

In abcd's own source checkout the person wants every abcd invocation, the plugin's hooks and command pages included, to run a build of the checkout's HEAD (dogfooding), and in every other repository to run the most recently installed stable release, so the two can always be compared. Today the hooks and the command pages resolve the plugin-root binary first, which is the last published release (v0.13.1 at 2026-10-07), so inside this checkout the guard, the loader and every slash command run stale code unless an agent knows to substitute go run ./cmd/abcd; the DOGFOODING rule in .abcd/rules.json teaches the agent but cannot reach the hooks. Automatic detection of abcd's source was rejected: a repository posing as abcd would get its own code run by the hooks, the hazard the hook shim's PATH rung already refuses (a binary inside the working tree).
