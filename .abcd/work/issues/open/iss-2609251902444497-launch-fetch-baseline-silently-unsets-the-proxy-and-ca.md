---
schema_version: 1
id: "iss-2609251902444497"
slug: "launch-fetch-baseline-silently-unsets-the-proxy-and-ca"
severity: "minor"
category: "ux"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/update/assets.go"
---

launch --fetch-baseline silently unsets the proxy and CA environment variables for the whole process: update.NewReleaseAssets reaches newUpdater, whose scrubEnv calls os.Unsetenv on HTTPS_PROXY, SSL_CERT_FILE and their kin, and the loud fetcher prints fetches only, while abcd update records the ignored names in its receipt. An operator behind a mandatory proxy sees a dial timeout with no hint, and a preview verb mutates its own process environment.
