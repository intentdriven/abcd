---
schema_version: 1
id: "iss-2608291941067275"
slug: "installer-failure-message-does-not-name-the-ignored-environment"
severity: "minor"
category: "ux"
source: "impl-review"
found_during: "ultra-v0.6.8-followup"
found_at: "site-src/install.sh.tmpl"
resolution: "the installer's fetch failure names the proxy, CURL_HOME and CA-bundle variables it ignores and says how to install by hand"
impact: fix
resolved_by:
  commit: "11240ef6"
---

ultra-v0.6.8 C4, the diagnostic half: the installer's fetch failure in site-src/install.sh.tmpl said only 'check the network', so on a host that reaches GitHub through HTTPS_PROXY or finds its CA bundle through SSL_CERT_FILE — both deliberately unset by the GHSA-x4v8-rxvx-8v89 lockdown — the cause was misdiagnosed. The message now names the variables the installer ignores (both cases of the proxy names) and says how to install by hand; the lockdown itself and whether to offer an escape from it stay with iss-2608291814562032.
