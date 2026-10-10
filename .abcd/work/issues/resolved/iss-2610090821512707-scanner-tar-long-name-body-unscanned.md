---
schema_version: 1
id: "iss-2610090821512707"
slug: "scanner-tar-long-name-body-unscanned"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "private security advisory GHSA-jx54-228j-mf98, filed 2026-10-05"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/container.go"
remedy: "Cover the bytes after the first NUL of a type-L or type-K long-link body on their own and keep the findings, a body that cannot be covered being Unscanned; prove it with a scanner test (watched fail first) that the reproduction archive hard-fails on the token or is Unscanned with the precheck returning ErrPayloadScanRefused, and a normal ustar archive of text still decodes; sweep siblings (other header bodies tar.Reader.Next consumes, such as PAX records)."
resolution: "decodeTar now walks the raw extension headers Next consumed: a GNU long name or long link body has its string covered as a field and the bytes after its NUL covered as a region, a PAX body is covered whole, and each body's padding must be zero."
impact: fix
---

The secret scanner reports a tar archive as content-decoded while the bytes after the first NUL of a GNU long-name (`././@LongLink`, type L) body are consumed by the tar reader and never scanned, so a gzip member there ships.

Private security advisory GHSA-jx54-228j-mf98 (draft, severity high). Full text, evidence and reproduction: the security-drain-2026-10-09 run directory in the main checkout's local tier. This record stays uncommitted until its fix lands; the fix commit adds it directly to resolved/.

Evidence (lines at main 7549ca2d5): `decodeTar` scans only what `archive/tar.Reader.Next` returns (internal/adapter/scanner/container.go:787); `Next` consumes the type-L body, keeps the C string before the first NUL as the next name and discards the rest. `coverTarHeader` scans `h.Name` only (internal/adapter/scanner/container.go:848), and the counting reader has already consumed the hidden bytes, so the trailer check misses them. `scanRefusals` treats ContentDecoded as covered and does not refuse ContentUnverified (internal/core/launch/dryrun.go:269), so returning not-decoded is not a fix. Distinct from GHSA-9wv7-88w3-f77m.

Reproduction: hand-build a ustar archive (Go's tar.Writer refuses a manual type-L header). First header `././@LongLink`, typeflag L, body `readme.txt`, NUL, then a gzip of long filler plus a `ghp_` token (not a literal in the raw archive). Second header a regular `readme.txt` of `harmless`. Two zero blocks. Scan beside README.md: ContentDecoded lists the tar and HardFails is 0; gunzip of the bytes after the NUL returns the token.
