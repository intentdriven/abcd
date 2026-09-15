---
schema_version: 1
id: "iss-2608291849371769"
slug: "scanner-text-branch-uncapped-and-gitignore-byte-branch"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "v0.6.9-security-pass"
found_at: "internal/adapter/scanner/scanner.go"
---

Launch payload scanner pre-existing debt: (1) the text branch of ScanBundle reads a bundle file with an uncapped os.ReadFile while the skip-listed branch is capped at maxBinaryScanBytes, so a large text file has no memory bound; (2) .gitignore sits on defaultSkipFilenames and therefore takes the byte-rule branch though it is text, so its prose/identity coverage is weaker than any other text file; findings also accumulate uncapped across files.
