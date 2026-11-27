---
id: dcn-2611270841518400
slug: scanner-callerhome-keeps-reading-one
date: 2026-11-27
---
2026-11-27 — scanner.CallerHome keeps reading one spelling of the home for now. The store sweep (SweepCallerHome) and the residual check that refuses a write (SurvivingCallerHome) take a single home string and read it through their own escaped views (JSON escapes, percent encoding), which fsutil.RedactHomeAt does not do; moving them onto the shared helper would drop those views, and teaching them several spellings changes the signature of every store redactor (memory, history, ideate, the privacy lint) and what their refusals count, which is more than a redaction fix should carry. What is lost: a home spelled with a doubled slash, a dot segment, a symlink, another Unicode form or the macOS data-volume path is not swept from those stores by the literal backstop, though the pattern detector still reads home-shaped paths. To reverse it, make CallerHome return the spellings fsutil.RedactHomeAt builds (exported from fsutil as a list) and loop SweepCallerHome and SurvivingCallerHome over them, longest first, in a change of its own with tests per store.
