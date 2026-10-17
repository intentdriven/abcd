---
schema_version: 1
id: "iss-2610151422164851"
slug: "at-docs-how-to-install-md-239-247-and"
severity: "nitpick"
category: "documentation"
source: "agent-finding"
found_during: "v0.14.2 release review, docs-currency pair, content commit 87513ff33"
origin: researcher-authored
production_mode: hand-written
found_at: "docs/how-to/install.md"
remedy: "Correct docs/how-to/install.md:239-247 and CHANGELOG.md:27 so it states what the shipped binary does, as the finding describes."
---

At docs/how-to/install.md:239-247 and CHANGELOG.md:27: Neither page says that a symlinked ~/.abcd.noindex (the dotfiles layout) never gets the .gitignore (homeignore.Lay returns NoHome), nor that an existing .gitignore there is kept as it is while the marker is written. install.md does say a symlinked folder is left as it is two lines earlier, so a careful reader can infer it. (v0.14.2 release docs-currency review, raised by sonnet.)
