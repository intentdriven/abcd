---
schema_version: 1
id: "iss-2609291925136841"
slug: "evaluate-whether-claude-md-can-be-removed-safely-now-that"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "rulings interview with the product thinker, 2026-09-29 (abcd-23 [b1e81b])"
origin: researcher-authored
production_mode: hand-written
remedy: "Inventory every CLAUDE.md consumer (harness load paths incl. sub-agents, abcd scaffolding, lint/positioning rules, docs); if all read AGENTS.md, remove CLAUDE.md here and stop scaffolding it in managed repos, with a test that fails if a consumer still needs it; otherwise record which consumer blocks it. Evaluated 2026-09-30 (research at .abcd/development/research/notes/2026-09-30-claude-md-consumers-and-agents-md-sota.md): not every consumer reads AGENTS.md, so CLAUDE.md stays for now. Blocking: lifeboat embark hard-codes CLAUDE.md as its marker target, and the harness reads no AGENTS.md before v2.1.277 (some session types before v2.1.281), with its agents-md plugin disabled, with the setting at claude-md, or when a CLAUDE.local.md sits in the tree. Removal needs: embark to follow docs.target like ahoy (test: embark into an AGENTS.md-only target), prepare-this-repo to stop scaffolding the symlink or scaffold an @AGENTS.md import, a harness version floor of v2.1.281 in the install checks, a conventions-router warning for CLAUDE.local.md beside AGENTS.md with no CLAUDE.md, then removal of the committed symlink with a fresh session shown loading AGENTS.md."
---

Evaluate whether CLAUDE.md can be removed safely now that Claude Code reads AGENTS.md natively. Today CLAUDE.md duplicates the AGENTS.md router. Check every place that relies on CLAUDE.md being present: the harness's own loading in the main session, sub-agents and plugin contexts; abcd's ahoy/prepare-this-repo scaffolding and lint rules that expect CLAUDE.md as a bridge; docs that name it; and managed repos abcd sets up. Only remove it if every consumer reads AGENTS.md.
