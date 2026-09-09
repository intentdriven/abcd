---
schema_version: 1
id: "iss-2609090951280114"
slug: "releaseof-boundary-still-admits-a-hyphen-suffixed-handle"
severity: "nitpick"
category: "bug"
source: "agent-finding"
found_during: "adversarial-review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/site/compose.go"
---

releaseOf now compares whole handles found by the package handle pattern rather than testing a substring, which stops a digit continuation such as itd-1990 from being read as a credit for itd-199. That pattern ends in a word boundary, and a hyphen is not a word character, so a hyphen-suffixed compound still yields the short handle as a match: run over the line renamed branch fix/itd-199-cleanup the pattern returns itd-199, and because the walk takes the newest dated section first, such a mention in a newer section out-stamps the real credit in an older one. Verified by running the shipped pattern over that line and over itd-1990, which correctly returns only the long handle. Dormant today, since the committed changelog carries no such compound token and nothing is mis-stamped now, and recorded because branch names, file stems and run ids of exactly that shape are ordinary changelog prose and the failure would be silent on the day one lands. Fix direction: require the character after the handle to be neither a word character nor a hyphen, or compare against the record ids the export already knows rather than against any handle-shaped token. Detector: a newer changelog section mentioning a hyphen-suffixed compound built on a handle must not stamp that handle, while a line naming the handle as itself still does. Residual of iss-2609081941074556.
