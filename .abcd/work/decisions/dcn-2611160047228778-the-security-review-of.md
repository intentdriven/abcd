---
id: dcn-2611160047228778
slug: the-security-review-of
date: 2026-11-16
---
2026-11-16 — The security review of itd-2610301914350541's final head d1ea70cec runs on Opus rather than Fable. The Fable reviewer hit its account limit, which resets on 2026-11-17 at 23:00, after approving 240e0daf1 in full over three rounds. d1ea70cec adds only a test-only commit, two string literals spelled through abcdhome.Display. A fresh Opus security reviewer reviews the whole branch at that head from the same brief, so the receipt is a full one; waiting about 46 hours for that delta would hold the intent's landing for no gain in assurance. The follow-up change that files this record and corrects the comment the Opus review flagged in questionLimitsAt is reviewed on Opus as well, for the same reason. To reverse: after the limit resets, run the Fable security reviewer at both merged commits and act on any finding as a follow-up.
