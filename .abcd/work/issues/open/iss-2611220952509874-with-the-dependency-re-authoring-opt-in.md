---
schema_version: 1
id: "iss-2611220952509874"
slug: "with-the-dependency-re-authoring-opt-in"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "autonomous fix run 2026-11-22"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/config/dependency-reauthor.conf"
remedy: "Add a command that relands a bot's dependency pull request as the person's own commit: it checks the pull request against the re-authoring bound in .abcd/config/dependency-reauthor.conf (bot, branch prefix, one commit, only the named manifests), applies the bot commit's change onto a fresh branch from origin's default branch, commits it under the checkout's git identity with the disclosure trailer when an agent runs it, and names the superseded pull request; and have the reauthor workflow's refusal and abcd's status name that command and the open bumps waiting on it."
---

With the dependency re-authoring opt-in present but its owner unset (and no app secrets), every Dependabot pull request fails the reauthor and attribution checks and stays open indefinitely: #1618 and #1619 sat for five days. Nothing reports the stuck bumps, and the only way to land one is by hand: redo the bump on the current default branch as the person's commit, within the bound the config declares, and let the bot close its pull request. A run on 22 Nov 2026 did exactly that, ad hoc.
