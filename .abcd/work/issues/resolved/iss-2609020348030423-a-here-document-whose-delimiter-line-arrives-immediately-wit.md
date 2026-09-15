---
schema_version: 1
id: "iss-2609020348030423"
slug: "a-here-document-whose-delimiter-line-arrives-immediately-wit"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/tokenize.go"
resolution: "Because the body is now consumed from the line immediately after the redirection, the FIRST delimiter line ends the document and every line after it reaches command position and is matched; a stray delimiter later in the input can no longer pull the commands between them into the body. Proved by the swallowed-hazard row of TestHeredocBodyStartsDespiteTrailingListOperator (internal/core/guard), watched failing on a scratch archive of the pre-fix tree where the same input returned allow. The silent allow fell out of the same change as its sibling, so it is resolved with it rather than separately."
impact: fix
---

A here-document whose delimiter line arrives immediately, with a stray delimiter at the end of the input, swallows the real commands between them as document body — a SILENT allow of a hazard bash runs, not a loud one (CWE-184, sibling of GHSA-5wx3-2c86-fjpx; pre-existing at v0.7.0). Evidence symbols: skipHeredocBodies and tokenize's newline branch in internal/core/guard/tokenize.go. Because the body was deferred while the redirection line ended in a list operator, the scan for the terminator started AFTER the early delimiter line and matched the stray one at the end, so `cat <<EOF &&` / `EOF` / `<hazard>` / `EOF` reported allow while bash runs the hazard (probed on bash 3.2: the hazard line runs, then bash reports `EOF: command not found`). The fix must consume the body from the line immediately after the redirection, so the first delimiter line ends the document and every line after it reaches command position and is matched.

## Grounds

- pursued: the swallow was the same defect seen from the allow side, and fixing where the body starts closes both; this would be shown wrong if the first delimiter line were not the one that ends the document
