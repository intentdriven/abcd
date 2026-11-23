---
id: dcn-2611231347142817
slug: six-refusal-codes-are-renamed-before-any
date: 2026-11-23
---
2026-11-23 — Six refusal codes are renamed before any release carries them, on the product thinker's acceptance of adr-2610301638558260 on 2026-11-23 after a check of how current tools name error codes: capture/remedy-missing is capture/missing-remedy, capture/input-invalid is capture/invalid-input, usage/bad-flag-value is usage/invalid-flag-value, capture/grounds-refused is capture/insufficient-grounds, capture/remedy-machine-value is capture/remedy-machine-specific, and explain/unknown-input is explain/unrecognized-input. They are renamed, not retired, because the ledger at v0.17.0 lists no codes, so the release gate passes and no replaced_by entry is owed. The tests that drive each refusal path now name the code as the string a reader matches on, so a later rename or a path that switches code fails there, which the ledger comparison cannot see (spc-2610301431108175, itd-2610301428378830).
