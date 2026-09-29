---
schema_version: 1
id: "iss-2609290541525428"
slug: "redactrefusal-does-not-seal-a-token-glued-behind-a-word-character"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/refusal.go"
---

scanner.RedactRefusal does not seal a token glued behind a word character. The scanner's bundled secret patterns anchor their start on a leading word boundary, and '_' and every letter or digit are word characters, so a token that sits right after one has no boundary and is never matched: the implement loop's laneFileGap (internal/core/implement/loop/receipt.go) returns the receipt's path as notes_ followed by a GitHub PAT and .md, does not exist, with the token raw, and memory's ValidateDistilledPage (internal/core/memory/schema.go) returns unknown key(s) [notes_ followed by the token] raw, while notes- and sub/ spellings seal. A letter on both sides (x, token, y) is missed the same way. Every RedactRefusal caller shares the gap (scribe, release, ideate, lifeboat, reading, intent, memory, implement loop), so a host payload's key or path carries a credential to the terminal and the transcript through a refusal. It is the refusal-side twin of the page-filename gap iss-2609290411321963 closed with filenameJudgeTexts. Found by the security review of lane drainEcho3. Fix direction: in the primitive, not the callers; re-find the secret patterns with their leading boundary removed, through the scanner's own linear adjacency machinery, and seal every glued match byte for byte through Redact; fail closed as before. Detector: a refusal naming a key or a path in which a well-formed token sits behind an underscore or a letter carries the token sealed.
