---
schema_version: 1
id: "iss-2609290656480443"
slug: "fsutil-readhomedeclaration-vets-the-declaration-file-s-own"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/fsutil/home.go"
resolution: "fsutil.ReadHomeDeclaration judges each directory below home on its opened descriptor and refuses one every account can write or another account owns; the group-writable case is the open question iss-2609290703091174"
impact: fix
resolved_by:
  commit: "57410aa48"
---

fsutil.ReadHomeDeclaration vets the declaration file's own mode and owner and refuses a symlinked directory on the way to it, but never vets the mode or owner of those directories: a ~/.abcd that every user can write (0777), or one another account owns, holding a 0600 file of the caller's still reads DeclarationOK. Anyone who can write that directory can rename or hard-link a file of the caller's shape in under a declaration's name (rules.json, trusted-roots, path-entry, credentials.json, config.json), so the leaf guards judge a file the caller did not put there. The guard rests on abcd's own writers creating ~/.abcd at 0o755/0o700; a directory made by hand, or re-moded, is never judged.

## Grounds

- pursued: no declaration is read through a directory another account can change except by group-write, which is deferred; a 0777 or foreign-owned ~/.abcd whose 0600 file still reads DeclarationOK would show it wrong
