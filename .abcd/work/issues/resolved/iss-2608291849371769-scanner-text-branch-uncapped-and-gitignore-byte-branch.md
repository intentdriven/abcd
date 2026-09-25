---
schema_version: 1
id: "iss-2608291849371769"
slug: "scanner-text-branch-uncapped-and-gitignore-byte-branch"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "v0.6.9-security-pass"
found_at: "internal/adapter/scanner/scanner.go"
resolution: "ScanBundle's text branch reads through fsutil.ReadGuarded under maxTextScanBytes (a file over it is an Unscanned gap with its reason); .gitignore is off the default skip filenames and takes the full text rules; findings are capped at maxBundleFindings with the remainder counted in findings_omitted and hard_fails counting every one."
impact: fix
resolved_by:
  commit: "ceb255225f91d2d3937aa8e0bb7fc088d06c76ba"
---

Launch payload scanner pre-existing debt: (1) the text branch of ScanBundle reads a bundle file with an uncapped os.ReadFile while the skip-listed branch is capped at maxBinaryScanBytes, so a large text file has no memory bound; (2) .gitignore sits on defaultSkipFilenames and therefore takes the byte-rule branch though it is text, so its prose/identity coverage is weaker than any other text file; findings also accumulate uncapped across files.

## Grounds

- pursued: every bundle read and the findings list are bounded without weakening the verdict; an oversized text file read whole, a .gitignore byte-scanned by default, or a hard fail lost to the cap would show it wrong
