---
schema_version: 1
id: "iss-2609251537550065"
slug: "fsutil-readdeclaration-judges-a-home-scoped-declaration-s"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/fsutil/fsutil.go"
---

fsutil.ReadDeclaration judges a home-scoped declaration's owner and permissions on an Lstat, then reads it through ReadGuarded's later open by path with no os.SameFile check between the two, so a co-tenant with write on the declaration's directory can rename-swap a different file in after the vetting and have it read as the caller's word; ReadGuardedInRoot already closes the same lstat-to-open window with os.SameFile. Callers: the rules user layer (~/.abcd/rules.json), trusted-roots, local-transcript-roots, path-entry, cache attestation and the implement limits file.
