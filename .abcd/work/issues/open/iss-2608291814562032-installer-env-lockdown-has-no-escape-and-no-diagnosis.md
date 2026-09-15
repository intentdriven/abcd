---
schema_version: 1
id: "iss-2608291814562032"
slug: "installer-env-lockdown-has-no-escape-and-no-diagnosis"
severity: "minor"
category: "ux"
source: "impl-review"
found_during: "ultra-v0.6.8-followup"
found_at: "site-src/install.sh.tmpl"
---

ultra-v0.6.8 C4 (capture only): site-src/install.sh.tmpl unconditionally unsets the proxy and CA-bundle variables (HTTPS_PROXY, ALL_PROXY, CURL_CA_BUNDLE, SSL_CERT_FILE, SSL_CERT_DIR, CURL_HOME) before any fetch. The lockdown IS the GHSA-x4v8-rxvx-8v89 fix and hooks/bootstrap.sh mirrors it, so it is deliberate; the cost is that a host whose curl finds its CA bundle only through SSL_CERT_FILE (NixOS, minimal containers, custom OpenSSL) or a network reachable only through HTTPS_PROXY cannot install, and the generic could-not-download message does not say why. The review proposes an explicit escape such as ABCD_INSTALL_KEEP_ENV=1; the objection is that an environment-variable opt-out reopens the very vector the lockdown closes (the poisoned environment sets the escape too). Whether to trade the lockdown for those users is a product decision, not a code fix. The purely diagnostic part — naming the ignored variables in the failure message — does not weaken the lockdown.
