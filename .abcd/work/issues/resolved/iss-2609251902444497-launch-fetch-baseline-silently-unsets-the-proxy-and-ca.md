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
resolution: "The release-asset fetcher ignores proxy and CA overrides for its own client only and leaves the process environment intact; the parity report, plain preview, pre-flight markdown and a failed fetch's refusal name every override that was set."
impact: fix
resolved_by:
  commit: "ab5f083f"
---

launch --fetch-baseline silently unsets the proxy and CA environment variables for the whole process: update.NewReleaseAssets reaches newUpdater, whose scrubEnv calls os.Unsetenv on HTTPS_PROXY, SSL_CERT_FILE and their kin, and the loud fetcher prints fetches only, while abcd update records the ignored names in its receipt. An operator behind a mandatory proxy sees a dial timeout with no hint, and a preview verb mutates its own process environment.

## Grounds

- pursued: with HTTPS_PROXY and SSL_CERT_FILE set, launch --dry-run --fetch-baseline leaves both set, connects directly and names both; a variable unset after the fetcher is built, or a preview that fetched without naming them, would show it wrong
