---
schema_version: 1
id: "iss-2608210923437502"
slug: "bundled-committing-domain-omits-the"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "memory-portability audit"
resolution: "The bundled COMMITTING domain gains two rules: commit as the human (never the tool's identity or a bot address), and re-read and strip a session URL or generated-with footer from a PR body, issue or comment after creating it."
impact: fix
resolved_by:
  commit: "2ee70951297579ecd1d86e51851cfa441450d9cd"
---

The bundled COMMITTING rules domain (the defaults every managed repo inherits from the binary) says Assisted-by-not-Co-Authored-By but omits the two attribution failures that keep recurring operationally and are currently patched by per-agent memory and per-routine prompt text: (1) the harness auto-appends a Generated-by footer plus session link to PR bodies and issue comments it creates — the domain should instruct the post-write read-back-and-strip (iss-178 remedy a, as an injected rule rather than every routine prompt re-carrying it); (2) autonomous runs default their git identity to the tool, which the attribution gate rejects (iss-193's class) — the domain should name the commit-as-the-human requirement. Prevention rung, complementing iss-178's detection remedy b; prompt-carried and memory-carried safety is the anti-pattern itd-114 just retired for minting

Siblings from the same day's parallel session (#404), which own the setup and gate mechanisms this capture deliberately does not duplicate: iss-2608210738367948 (ahoy/new-repo setup writes the routine git identity), iss-2608210738363966 (mirror the attribution gate into pre-push). This capture is the third, complementary rung — the INJECTED-RULE prevention: the bundled COMMITTING domain should carry the footer-strip-after-gh-writes line (and a pointer to the identity requirement) so every managed repo's agents receive the instruction per-prompt without any routine prompt or per-user memory re-carrying it.

## Grounds

- pursued: a managed repo's agents now receive both attribution rules per prompt without routine-prompt or memory carriage; a prompt matching COMMITTING that injects neither rule would show it wrong
