---
schema_version: 1
id: "iss-2610140720357027"
slug: "termsafe-sanitize-leaves-the-variation"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "the security review round 2 of fix/sec-build-next-series"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/termsafe/termsafe.go"
remedy: "mask every listed character but the variation selectors outright; mask a variation selector only in a run of two or more, or after a base with no emoji or standardized variation sequence for it (Unicode's emoji-variation-sequences.txt and StandardizedVariants.txt), so heart plus VS16 and the keycap digits survive, with a table test of legitimate emoji sequences beside the masked ones"
---

termsafe.Sanitize leaves the variation selectors, about fifty other invisible format characters and the Hangul fillers unmasked. The variation selectors U+FE00-U+FE0F and U+E0100-U+E01EF, the Cf characters U+206A-U+206F, U+1D173-U+1D17A, U+13430-U+1343F and U+110BD, the Mongolian free variation selectors U+180B-U+180D, the Hangul fillers U+115F, U+1160, U+3164 and U+FFA0, the Braille blank U+2800, the line and paragraph separators U+2028 and U+2029 and the object replacement character U+FFFC each render as nothing or as blank space, so two byte strings can display identically and a run of selectors can carry a hidden payload past a reader (the Opus and Sonnet security reviews of fix/sec-build-next-series, round 2). They were not added blind: VS16 (U+FE0F) is legitimate in emoji, a heart followed by VS16 being the common case, and masking it would put a ? into ordinary text a repository writes.
