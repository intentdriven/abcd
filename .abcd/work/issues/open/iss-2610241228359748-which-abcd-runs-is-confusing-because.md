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
