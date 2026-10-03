---
schema_version: 1
id: "iss-2610020704163406"
slug: "the-comment-line-of-the-gitignore-block-ahoy-install-writes"
severity: "nitpick"
category: "ux"
source: "managed-repo"
found_during: "peer report: ahoy install --adopt on a private consumer repo (abcd v0.9.0), reproduced at 7fb52a6b5"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/gitignore.go"
remedy: "Write the managed comment in plain ASCII (for example '# abcd-managed block, do not hand-edit; run /abcd:ahoy to refresh.'), keeping the BEGIN/END fence markers detection reads unchanged, and accept the old comment as the same block so existing installs do not read as drift; test: every byte the install writes into .gitignore is ASCII."
---

The comment line of the .gitignore block ahoy install writes into an adopted repository uses an em dash ('# abcd-managed block — do not hand-edit. Run /abcd:ahoy to refresh.'), so a repository whose own writing rules ban em dashes or non-ASCII in committed files fails them on a line it cannot edit, since the block is managed. Reproduced at tip with a private-visibility install into a scratch repository. Sibling of the conventions-file block's em dashes (iss-2610020700529281).
