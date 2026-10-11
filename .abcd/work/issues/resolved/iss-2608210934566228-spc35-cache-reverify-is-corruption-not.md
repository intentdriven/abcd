---
schema_version: 1
id: "iss-2608210934566228"
slug: "spc35-cache-reverify-is-corruption-not"
severity: "critical"
category: "security"
source: "user-observation"
found_during: "spc-35 adversarial review 2026-08-21"
found_at: "hooks/bootstrap.sh"
resolution: "cache promotion authenticates the cached sha against the release's PUBLISHED checksums.txt before trusting it (online resolve answering the cached tag); offline stays corruption-only with an honest notice naming the trust level; includes the self-consistent poisoned-pair watched-fail test"
impact: fix
resolved_by:
  commit: "c9cf9e19beada6f6d7f3bb077160eec65002d5b2"
---

spc-35 BLOCK (security review): the cache promotion 're-verify against recorded binary_sha256' is a corruption check, not a tamper check. The cached artefact and its binary-meta (holding the expected hash) both live in ${CLAUDE_PLUGIN_DATA}/cache, equally same-UID-writable. An attacker writing BOTH — payload plus a meta recording its hash and the current release_tag — passes the promotion gate and gets an unverified binary installed at the guard path and onto PATH, with the success notice asserting 'checksum-verified'. Demonstrated end-to-end vs live v0.6.1. This is exactly adr-46's central claim: on main a poisoned plugin-root binary healed at the next update (harness re-clones a fresh root -> bootstrap re-downloads+verifies from GitHub); here the poisoned cache is preferred over the network indefinitely and survives every update. adr-46, invariant 12, and the notice assert a protection the code does not provide. Fix: in cache mode (which already resolves the latest tag over the network) GET that release's checksums.txt and verify cached_sha against the PUBLISHED digest before use_cache=yes — restores tamper-evidence online, offline stays corruption-only (document honestly). Plus the missing test: TestBootstrapCorruptCacheRefusesLoudly rewrites only the artefact, never the meta, so it never exercises the case that succeeds.