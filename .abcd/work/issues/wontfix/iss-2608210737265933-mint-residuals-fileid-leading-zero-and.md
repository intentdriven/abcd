---
schema_version: 1
id: "iss-2608210737265933"
slug: "mint-residuals-fileid-leading-zero-and"
severity: "nitpick"
category: "security"
source: "impl-review"
found_during: "itd-114 pre-PR security review"
wontfix_reason: "accepted mint residuals recorded deliberately so future hunts do not re-file them (spc-33 bounds-and-residuals); neither has an attack path under current wiring"
---

Two accepted mint residuals recorded so future hunts do not rediscover them as findings (spc-33 bounds-and-residuals section; surfaced by the pre-PR security review of the itd-114 delivery): (1) recordid's fileID rebuilds an id from the parsed integer, so a timestamp stamp with a leading zero — reachable only from a host clock in 2000-2009 or from 2100 onward — resolves to a different id than its filename claims (the year-2100 horizon the spec accepts); (2) the mint's rejection-sampling loop is uncapped, safe today because entropy is crypto/rand or an injected test reader that EOFs, but if Minter.Entropy ever becomes configurable from env/config/flags the loop needs an iteration cap. Neither has an attack path under the current wiring