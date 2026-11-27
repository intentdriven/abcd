---
schema_version: 1
id: "iss-2611252118365874"
slug: "the-guard-reads-a-go-file-handed-to-go"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "guard review follow-up 1125"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/script.go"
remedy: "Read no file a go run names as a shell script: leave a .go word behind the go run runner to the reading of the program it builds, and test that a Go file with an apostrophe in a comment draws no verdict."
---

The guard reads a Go file handed to go run by its path as a shell script. The file stands in command position behind go run, and its name holds a slash, so the script reading opens it and splits its Go text into shell words; a comment or string holding an apostrophe is then an unterminated quote, and go run ./cmd/abcd/main.go version is refused as command-unparsable. A file named with no slash (go run main.go) is not read. Seen on 2026-11-25 with this repository's cmd/abcd/main.go and internal/core/guard/doc.go.
