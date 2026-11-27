---
id: dcn-2611262047043844
slug: capture-owed-answers-a-repository-with
date: 2026-11-26
---
2026-11-26 — capture owed answers a repository with no commit as an empty list, and refuses a put or a settle there with capture/invalid-input, since the list is keyed on the root commit and there is none; a store it cannot open safely, or whose lock it cannot take in time, is a fault (exit 1) naming the store with the home written as ~, as spc-2611252115253843 adds no refusal code. Grounds: the spec's no-new-code rule and the store's own refusals. Reverse: give contention its own exit 3 code in internal/core/refusal.
