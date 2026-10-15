---
id: dcn-2609300000000021
slug: an-older-site-interface-string-file
date: 2026-09-30
---
2026-09-30 — An older site interface-string file keeps building: `abcd site setup` and `abcd site build` add to `site-src/ui.json` each label the allowlist declares and the file does not carry, with abcd's default words, name each on stderr, and change nothing else in it (the product thinker's ruling TG1 of 2026-09-30, relayed verbatim: "(b) ABCD ADDS THE MISSING LABELS: on the next site setup or site build, abcd adds only the missing required labels (with the default words); the project's own wording elsewhere in ui.json is never changed. No failure, no manual step; both intents stay impact: additive."). It is the one exception to "a file the repository owns once it exists is kept", recorded as adr-2609301720596683, which refines adr-47 and leaves decision 2's closed allowlist untouched: a blank declared label and an unknown key are still refused, and the site gate's own render never completes the file. itd-2609212103568351 and itd-2609212103572513 keep `impact: additive` (lane tgLabels of autonomous run A).
