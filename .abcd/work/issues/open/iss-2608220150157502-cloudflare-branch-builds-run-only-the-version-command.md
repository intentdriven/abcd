---
schema_version: 1
id: "iss-2608220150157502"
slug: "cloudflare-branch-builds-run-only-the-version-command"
severity: "minor"
category: "drift"
source: "user-observation"
found_during: "abcdev-site-plan investigation 2026-08-21"
found_at: "wrangler.jsonc"
deferred_after: "v0.9.0"
deferral_reason: "Ruled by the product thinker at the 2026-09-23 run A interview (M12: the host dashboard is authoritative for the build settings (wrangler.jsonc now says so); a check that reads the real setting is ruled and needs planning)."
---

Cloudflare non-production branch builds run only the version command, so branch previews repeat the production build instead of building the branch — observed and recorded as comments in wrangler.jsonc. The adr-48 deploy design replaces these with a labelled preview deployed from Actions on push to main and turns Cloudflare's automatic production builds off