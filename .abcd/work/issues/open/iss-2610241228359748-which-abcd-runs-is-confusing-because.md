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
