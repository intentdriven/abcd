---
schema_version: 1
id: "iss-2611220956420939"
slug: "naming-the-owner-whose-authorship"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "autonomous fix run 2026-11-22"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/config/dependency-reauthor.conf"
remedy: "Make it a setup step: ahoy offers to name the owner from the person's own git identity (user.name and user.email), asks them to confirm, and stores the values outside the tree, as repository Actions variables set through the forge (which the workflow reads as vars.*) since a workflow on the forge cannot read a file on one person's machine; the workflow falls back to the committed conf only where the variables are absent, and the same step walks the person through the App and its two Dependabot secrets."
---

Naming the owner whose authorship dependency re-authoring asserts is no step of abcd's per-repository setup: a repository that opts in ships owner_name and owner_email empty, and every bot bump is refused until a person finds the config and edits it by hand, as happened on 22 Nov 2026. The values also live in a committed file, though they are one person's acceptance rather than a property of the repository.
