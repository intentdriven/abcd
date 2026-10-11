---
schema_version: 1
id: "iss-2610090821499579"
slug: "scanner-png-chunk-tail-after-zlib"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "private security advisory GHSA-83x2-mf5v-j796, filed 2026-10-05"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/container.go"
remedy: "Return the unread suffix from `inflate` and cover it on its own in every chunk caller (zTXt, compressed iTXt, iCCP, IDAT), a suffix that cannot be covered being Unscanned; prove it with a scanner test (watched fail first) that the reproduction PNG hard-fails or is Unscanned, a PNG whose zTXt is only the zlib member still decodes, and a token inside inflated IDAT pixels stays the documented residual; sweep siblings (every caller of inflate)."
resolution: "inflate now returns the bytes a PNG chunk carries after its zlib stream, and every compressed chunk (zTXt, iTXt, iCCP, IDAT) covers that tail as a region, so a member hidden there is decoded and scanned."
impact: fix
---

The secret scanner reports a PNG as content-decoded while bytes inside a compressed chunk after the zlib checksum are never inflated or scanned, so a gzip member there ships through the launch gate.

Private security advisory GHSA-83x2-mf5v-j796 (draft, severity high). Full text, evidence and reproduction: the security-drain-2026-10-09 run directory in the main checkout's local tier. This record stays uncommitted until its fix lands; the fix commit adds it directly to resolved/.

Evidence (lines at main 7549ca2d5): `decodeBudget.inflate` uses `zlib.NewReader`, which stops at the Adler-32, and returns only the inflated bytes (internal/adapter/scanner/container.go:236). The `zTXt` arm covers only that output (internal/adapter/scanner/container.go:970) and `decodePNG` returns decoded after IEND (internal/adapter/scanner/container.go:951). `decodeStream`, the top-level zlib path, does cover the unread tail (internal/adapter/scanner/container.go:329). The same unread tail exists for compressed `iTXt`, `iCCP` and `IDAT`. Distinct from GHSA-9wv7-88w3-f77m (the skip that never opened the PNG). A PNG-only bundle is refused by the zero-coverage sentinel; the bypass needs one other full-text file, which the include list already has.

Reproduction: build a small valid PNG and insert a CRC'd `zTXt` chunk before IEND: keyword `Comment`, NUL, compression method 0, a zlib member of `harmless` plus a newline, then a Huffman-coded gzip member of a `ghp_` token. Scan it beside README.md: `ContentDecoded` lists the PNG and `HardFails` is 0; gunzip of the tail returns the token.
