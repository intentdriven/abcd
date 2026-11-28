---
schema_version: 1
id: "iss-2610241228359748"
slug: "which-abcd-runs-is-confusing-because"
severity: "major"
category: "process"
source: "user-observation"
found_during: "/abcd:abcd after a plugin update to v0.15.1, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: "plugin install scopes and the binary resolution ladder"
remedy: "Rule in one ADR a single precedence for which abcd runs across plugin scopes (project and user), the plugin-root binary, the PATH binary and a source build, and make the board, ahoy and every command page name the copy that applies and report any other installed copy whose version disagrees."
deferred_after: v0.15.1
deferral_reason: "Which copy of abcd runs when several are installed needs the product thinker's design ruling (an ADR on precedence across plugin scopes, the PATH binary and a source build) before any code fix; behaviour is unchanged since v0.15.1 (ruled 27 October 2026)."
---

Which abcd runs is confusing because several installs coexist and nothing says which one wins: the plugin can be installed at user scope and at project scope at once, each with its own plugin-root binary, and there is also a PATH binary (ahoy install) and, in abcd's own checkout, a source build (go run). Observed 2026-10-24: the person updated the plugin to v0.15.1, but the update landed as a new project-scope install for this repository while the user-scope install stayed at 0.13.3, and the session went on loading the user-scope copy (command pages still named .../abcd/0.13.3/abcd, its hooks bootstrapped there at 14:23); the 0.15.1 directory had never bootstrapped and held no binary; and the binary inside the 0.13.3 directory reported v0.14.1, so even the directory name did not say which version ran. Nothing in the board, ahoy or the command pages surfaced any of this. The person wants it unified (for example: does a repository-scope plugin win over a user-scope one, and how does the CLI on PATH relate to either?). Earlier rulings each settled one rung on its own (dcn-2609080000000001 the owned-only PATH rung for the hook shims, adr-2609151706587280 the cache-to-PATH attestation, iss-2610070547057816 dogfooding a source build in abcd's own checkout) without addressing the underlying problem: no single, stated precedence across plugin scopes, the plugin-root binary, the PATH binary and a source build.

## Deferral 2026-10-27

Deferred past v0.15.1: Which copy of abcd runs when several are installed needs the product thinker's design ruling (an ADR on precedence across plugin scopes, the PATH binary and a source build) before any code fix; behaviour is unchanged since v0.15.1 (ruled 27 October 2026).

## Evidence 2026-11-21

The same split recurred. On one machine the plugin is installed at user scope at v0.16.0 (updated 2026-10-31, the only copy holding a binary) and at project scope for this repository at v0.16.2 (2026-11-06), while the newest release is v0.17.0 (2026-11-18). Sessions started in this repository on 2026-11-20 were served the user-scope copy, its command pages and its hooks. Three things followed in one session.

- The `/abcd:ideate` page it was handed predates the current-state mode, which the v0.16.1 and v0.16.2 pages carry. The session ran the recorder built from the tree, as the rule for this checkout requires, and that recorder refused the verdict for its missing `current_state` after all three legs. The adversarial leg was run again; a second session met the same refusal the same day.
- The page named the v0.16.0 binary for the record step. That release has no `current_state` field and refuses a document carrying one as an unknown field, so a verdict the page's own binary accepts is one the tree's refuses, and the reverse.
- The rules hook injected between 20 and 53 thousand characters on each of four prompts over 20 and 21 November, against the 10,000-character cap of the host, which abcd stays under from v0.17.0 (iss-2611041419354106), so the host kept all but a preview of about two thousand characters out of the session.

`abcd ahoy` from the tree reported no gaps. At a session start on 21 November the hook's one line in the session was a count: two notices on the hook's stderr, with `abcd history staged` and `abcd ahoy` named as where to look. The text of the notices did not reach the session, so what they said was not seen. The hook's code composes a notice at that point for a binary that is behind its own source (internal/surface/cli/staleness.go), and its advice is to rebuild with `make build`, which does not fit a binary in a plugin's folder. Asked directly, the installed copy's `--version` said it was behind the checkout tip. The part of the ideate failure that a current page would not cure is iss-2611211024539425 and iss-2611211025072947, and the missing notice of a newer release is iss-2610290709017008.

2026-11-25: The ideate verdict dcn-2611251807556458 rests on this: agreement among the copies is the durable core, whatever is ruled on checking for a release.

## One abcd per account

One answer proposed for the plugin-root-against-PATH rung of the precedence this record asks one ADR to rule is to install abcd once per account (the owned PATH copy) and have the plugin run that copy, so that every session on a machine runs the same release. It is part of this record's question, not a separate problem.

What holds today:

- **The host's choice is deterministic.** The host keeps one cache directory per installed plugin version (`cache/<marketplace>/<plugin>/<version>/`), points `CLAUDE_PLUGIN_ROOT` at the one a session loaded, and keeps using it for that session's hooks until `/reload-plugins`. It removes an updated-away directory only 14 days after marking it orphaned, so a session that loaded the old version keeps running. Nothing tells a session which root it holds or that a newer one is installed. Which install a session loads when the plugin is installed at user and at project scope at different versions is not documented, so that half of this record stays a question for the ADR.
- **Two releases ran side by side on one machine, 2026-11-25 to 26.** The cache held 0.16.0, 0.16.2, 0.17.0 and 0.17.1 side by side, and only 0.16.0 and 0.17.1 held a binary. A session started on 0.16.0 went on running hooks from the 0.16.0 root after 0.17.1 was installed on 25 November, and on 26 November its Stop hook still cut notes at 256 bytes, which b46125db7 (in v0.17.1) raised to 512. The owned PATH copy on the same machine was already at 0.17.1, because the bootstrap refreshes it on each new release and the newest-release mark only lets it move forward. So two concurrent sessions on one machine ran different abcd releases against the same stores, while a copy that carried the fix was already installed.
- **An account copy is already a rung, as a fallback only.** Each hook shim but SessionStart's runs `${CLAUDE_PLUGIN_ROOT}/abcd` when it exists and reaches the PATH copy only when the plugin root holds no binary, and only when `~/.abcd.noindex/path-entry` records that exact path (dcn-2609080000000001); SessionStart has no PATH rung and fails closed. Where the person has listed the checkout in `~/.abcd.noindex/dogfood-roots`, a build of the reviewed branch's tip (`hook dogfood-binary`) runs ahead of the root binary in each shim.

Why the plain form, one binary per account serving every session, fails:

- **It moves the skew instead of removing it.** The host pins a session's command pages and `hooks.json` to the root the session loaded, so a forward-only account binary pairs every older session's pages and hooks with a newer binary: the 2026-11-21 shape above, where an older command page met a newer recorder that refused what the page produced, and the plausible-wrong-answer shape of iss-2610141647314635.
- **It reopens what itd-132 ruled out.** itd-132 kept execution in each plugin root and ruled out a shared mutable binary across roots for skew semantics, cross-root locking, dev-shim collision and an at-rest tampering window. Reopening that takes an ADR that supersedes it and answers each of the four reasons. That ruling's premise is already weaker than it reads: `internal/surface/cli/skew.go` notes that one cached binary now serves every plugin root, and the 2026-10-24 observation above found a 0.13.3 directory holding a v0.14.1 binary.
- **Trust exposure widens.** The PATH rung compares a recorded path and never hashes. As the main rung it would carry every PreToolUse guard decision, out of a user bin directory that other installers also write to, and another project also ships a command named `abcd` (https://github.com/libAtoms/abcd). The posture against a hostile process running as the same user does not change, but the exposure to an accidental overwrite does. A cheap check once per session (the recorded hash at SessionStart, or the release against the loaded surface's version) answers it.

A proposal for this record's ADR to weigh: the account (owned PATH copy, attested cache, newest-release mark) is the one authority on which release is current. Its copy serves a session's hooks only when its release matches the plugin surface that session loaded. On a mismatch the session's own root binary runs, and the session is told, at its start and in its hooks, which copy ran, which copy is newer, and to run `/reload-plugins` (iss-2610290709017008). The ADR still has to rule the user-against-project scope question, because one binary does not decide which pages and hooks a session loads.

What that proposal implies building: an ADR superseding itd-132's per-root execution to that extent. A shim ladder that puts the owned PATH copy first when its release matches the surface, and falls back to the root binary otherwise, which in SessionStart means adding a PATH rung where it now fails closed on purpose. A stated compatibility window between surface and binary that covers every vintage the host can still load, with refusals that say so rather than plausible answers. The board, ahoy and every command page naming the copy that ran and any copy that disagrees. A rule for what moves the account copy and how a person pins or rolls it back for all sessions at once. And behaviour for an install with no owned PATH copy, plus uninstall semantics for the shared copy.
