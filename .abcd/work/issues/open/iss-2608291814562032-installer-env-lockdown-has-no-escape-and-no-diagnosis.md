---
schema_version: 1
id: "iss-2608291814562032"
slug: "installer-env-lockdown-has-no-escape-and-no-diagnosis"
severity: "minor"
category: "ux"
source: "impl-review"
found_during: "ultra-v0.6.8-followup"
found_at: "site-src/install.sh.tmpl"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker on the escape, which the record already names as a product decision: the diagnostic half is done (the public installer names the ignored variables since 11240ef6a, and the bootstrap hook's two network refusals name them in this lane), and the lockdown itself is unchanged. Whether a proxy-only or custom-CA host gets a way through, and by what, is open (drain lane drainRest, run A, 2026-09-29)."
remedy: "Waits on the owed ruling (a way through for proxy-only or custom-CA hosts, or none): if an escape is granted, take it as an explicit argument to the public installer (sh -s -- --proxy URL --cacert FILE) passed to curl as its --proxy and --cacert options while the environment lockdown stays, and never offer it in the unattended bootstrap hook; if not, document the lockdown and a manual download-and-verify route in the install docs. Prove the escape with an installer test in which a poisoned HTTPS_PROXY is still ignored and the argument is honoured."
---

ultra-v0.6.8 C4 (capture only): site-src/install.sh.tmpl unconditionally unsets the proxy and CA-bundle variables (HTTPS_PROXY, ALL_PROXY, CURL_CA_BUNDLE, SSL_CERT_FILE, SSL_CERT_DIR, CURL_HOME) before any fetch. The lockdown IS the GHSA-x4v8-rxvx-8v89 fix and hooks/bootstrap.sh mirrors it, so it is deliberate; the cost is that a host whose curl finds its CA bundle only through SSL_CERT_FILE (NixOS, minimal containers, custom OpenSSL) or a network reachable only through HTTPS_PROXY cannot install, and the generic could-not-download message does not say why. The review proposes an explicit escape such as ABCD_INSTALL_KEEP_ENV=1; the objection is that an environment-variable opt-out reopens the very vector the lockdown closes (the poisoned environment sets the escape too). Whether to trade the lockdown for those users is a product decision, not a code fix. The purely diagnostic part — naming the ignored variables in the failure message — does not weaken the lockdown.

## Remedy grounds (2026-09-29)

- The objection to ABCD_INSTALL_KEEP_ENV=1 is that a poisoned environment sets it too; an argument the person types is not ambient state, which answers it. rustup and curl both read proxies from the environment by default: https://rust-lang.github.io/rustup/network-proxies.html and https://everything.curl.dev/usingcurl/proxies/env.html (consulted 2026-09-29), the behaviour GHSA-x4v8-rxvx-8v89 had to close.
- Rejected: the environment-variable escape the review proposed, for the reason above.
