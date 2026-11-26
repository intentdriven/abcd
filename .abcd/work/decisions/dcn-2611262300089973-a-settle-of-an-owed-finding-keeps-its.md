---
id: dcn-2611262300089973
slug: a-settle-of-an-owed-finding-keeps-its
date: 2026-11-26
---
2026-11-26 — A settle of an owed finding keeps its reason only when it is typed in the checkout the entry was put from; from any other checkout the act is kept and the reason withheld. Grounds: the spec redacts a reason with the entry's own checkout's patterns, and building that redactor reads that checkout's scanner configuration, which the person settling from elsewhere has not looked at and may have changed since the put; withholding keeps the spec's redaction rule and reads only the caller's checkout. The other way, redacting the reason with the caller's own checkout, would keep every reason but drop the spec's rule and the withholding cases built on it. Reverse by redacting with the caller's checkout instead, in the findings store's settle, and editing the spec's sentence on withheld reasons.
