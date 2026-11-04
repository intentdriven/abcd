---
schema_version: 1
id: "iss-2611040554213730"
slug: "06-capture-md-296-says-a-flag-value"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/06-capture.md"
remedy: "Validate --lapsed-at as a flag before the write so it exits 2, with a test."
---

06-capture.md:296 says a flag value outside its shape exits 2 on the capture write, and 06-capture.md:143 says a lapse instant that is given must be an RFC 3339 instant. abcd capture "<text>" --remedy r --lapsed-at bad is refused with malformed frontmatter: lapsed_at is not an RFC 3339 instant, and exits 1. A bad --severity and a --found-at path that does not exist exit 2.
