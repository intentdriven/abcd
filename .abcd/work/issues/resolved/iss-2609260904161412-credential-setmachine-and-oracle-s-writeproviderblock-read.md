---
schema_version: 1
id: "iss-2609260904161412"
slug: "credential-setmachine-and-oracle-s-writeproviderblock-read"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/credential/credential.go"
resolution: "Both setup writes hold fsutil.WithFileLock across the read, the change and the rename (.credentials.json.lock and .config.json.lock beside their files), and the provider block is re-checked under the lock so a second concurrent setup of one provider is refused."
impact: fix
resolved_by:
  commit: "4762bc2c"
---

credential.SetMachine and oracle's writeProviderBlock read, modify and rename ~/.abcd/credentials.json and ~/.abcd/config.json with no lock, so concurrent ahoy connect runs lose a key or a provider block while each reports it wrote the file.

## Grounds

- pursued: concurrent ahoy connect runs keep every key and every block, and exactly one of several setups of one provider succeeds; shown wrong by a writer of either file that does not take its lock, or by TestConcurrentConnectsKeepEveryKeyAndBlock or TestConcurrentSetsKeepEveryEntry losing an entry
