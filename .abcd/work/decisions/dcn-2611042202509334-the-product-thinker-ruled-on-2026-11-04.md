---
id: dcn-2611042202509334
slug: the-product-thinker-ruled-on-2026-11-04
date: 2026-11-04
---
2026-11-04 — The product thinker ruled on 2026-11-04 that the merge queue runs Linux on every entry and hosted macOS runs nightly and on demand, after a day on which queue entries kept dropping because GitHub had no macOS runner to give, with no test failing. Each release cut also runs its own macOS suite before releasing, whatever the nightly said. The release path runs it: release.yml's tag and publish jobs need a gate over the shards it runs on the commit it is about to tag (iss-2611042224527148). A self-hosted runner is held for later, if one is ever needed.
