---
schema_version: 1
id: "iss-2611271406524430"
slug: "library-ledger-reads-flip-0-as-no-flip"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/source.go"
remedy: "tell a given --flip apart from an absent one with the flag's Changed state, and refuse a value below 1 at exit 2 naming --flip and that it takes a ledger line number, 1 or more"
resolution: "library ledger now tells a given --flip from an absent one by the flag's Changed state, and refuses a value below 1 at exit 2 naming --flip and that it takes a ledger line number, 1 or more, without repeating the value typed; TestLibraryLedgerFlipBelowOneNamesFlip covers 0, --flip=0 with --json, and a negative"
impact: fix
---

library ledger reads --flip 0 as no flip given, so it falls through to the append path and refuses with the bare 'nothing to append' message instead of naming --flip and its valid range
