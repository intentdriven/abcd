---
schema_version: 1
id: "iss-2610241423411800"
slug: "on-a-new-machine-or-account-the-private"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "autonomous run, 2026-10-24 (product thinker)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/banlist/private.go"
remedy: "Derive the private layer's machine defaults by rule at guard time, never stored: enumerate local accounts (uid at or above the first user uid), the host's computer, local and DNS names, interface MACs, and the names a tailnet or local resolver lists, and match accounts only in context (home path, login@host) with the scanner's identity rules (internal/adapter/scanner) as the one canonical home; run the same set before each test run and in the guard, and have abcd setup or install report the derived set by key; the stored file keeps only what no rule can find."
---

On a new machine or account the private banlist starts empty: install lays the store's header and no entries, and the pre-commit guard then checks nothing while printing only a warning. On 2026-10-24 a new account's guard checked no entries during a day of autonomous commits from seven sessions. The defaults it should hold are known in advance: this machine's names (computer name, local host name, the tailnet domain, interface MAC addresses), every local user account, and the names of the machines on its network. The person wants them collected by rule at each commit and each test run, not stored as a word list, so a new account, machine or device is covered without an edit. A hand-rolled prefill of seven keyed entries was written on this machine to test the guard; it refused a canary commit by key. Bare generic account names (admin, dev) cannot be word patterns: dev matches development, and the scanner's tests use [redacted-path] and dev@host as fixtures, so accounts must be matched in context (a home path, login@host).

## Security assessment (abcd fixes what it adds)

This is not recorded as a security finding. The route exists without abcd: a person can commit their own hostname or account name to any repository whether abcd is installed or not. The private banlist is a guard abcd adds against that, and an empty default is a missing convenience in that guard, not a vulnerability abcd introduced. So under .abcd/development/principles/abcd-fixes-what-it-adds.md it is a follow-up to improve the guard's defaults, filed as the product thinker asked ("worth capturing as a follow-up"), and it needs no private advisory.
