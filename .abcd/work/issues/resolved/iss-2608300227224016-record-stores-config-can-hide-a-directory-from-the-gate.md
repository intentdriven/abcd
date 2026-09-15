---
schema_version: 1
id: "iss-2608300227224016"
slug: "record-stores-config-can-hide-a-directory-from-the-gate"
severity: "major"
category: "security"
source: "impl-review"
found_during: "itd-180 adversarial security review, 2026-08-30"
found_at: "internal/core/lint/schema.go (nestedStoreRoots), internal/core/lint/config.go"
resolution: "The nested-store-root exemption is now derived from the code-side recordStores looked up in the config, and parseConfig refuses a record_stores key naming no store — which closed the unknown-prefix half only: a KNOWN prefix aimed at another store's declared bucket did the same damage until iss-2608300320001985."
impact: fix
---

record_schema's nested-store-root exemption is derived from every value in the record_stores config map, including keys naming no store the scanner knows, so a committed config line such as an unknown prefix pointing at .abcd/work/issues/anything hides that directory from the undeclared-bucket blocker without anything scanning it; the file's own comment says a config that could add a bucket could also hide one and that the list is code, not config. Build the nested set from the code-side recordStores looked up in the config, and refuse unknown record_stores keys at parse.
