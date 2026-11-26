---
id: dcn-2611260007036461
slug: a-fourth-fix-round-on-the-sub-agent-hook
date: 2026-11-26
---
2026-11-26 — A fourth fix round on the sub-agent hook's branch, past the three-round cap, for one security finding only: the cap exists to stop review loops, and the security rule requires a finding raised in review to be fixed before its change lands. The round carries nothing else; any further finding is fixed or deferred as its own record. Reverse: none needed once the round lands.
