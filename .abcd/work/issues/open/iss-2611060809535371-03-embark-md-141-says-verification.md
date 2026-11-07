---
schema_version: 1
id: "iss-2611060809535371"
slug: "03-embark-md-141-says-verification"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/03-embark.md"
remedy: "Say the symlink refusal covers the whole tree and the size cap covers the hashed files, or apply the cap to the excluded layer too."
---

`03-embark.md:141` says verification refuses a symlink or an oversize file anywhere inside the lifeboat. The size cap applies only to files the manifest hashes (`internal/core/lifeboat/embark.go:218`): an 18 MiB `review/big.bin` in the excluded layer passes `abcd embark probe` with exit 0, while a symlink there and an oversize hashed file are refused. The comment at `internal/core/lifeboat/embark.go:184` carries the same wording.
