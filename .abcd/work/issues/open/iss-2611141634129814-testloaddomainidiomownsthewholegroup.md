---
schema_version: 1
id: "iss-2611141634129814"
slug: "testloaddomainidiomownsthewholegroup"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "a merge queue removal"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/rules/load_test.go"
remedy: "In the test's script (internal/core/rules/load_test.go), report on each of the two paths that exit 4 which check ended it and what it read: the pid's value where the pid-shape check (isPID) fails, and what ps returned where the parent check (ownChild) fails, so a later failure names its cause. The script points its own stderr at the null device before the launch (detach), so the report goes to stderr only through a descriptor kept open for it, or the two checks take different exit statuses. Also read the child's parent with a short bounded retry (a few readings some tens of milliseconds apart) before the parent check exits. The launch text the rule teaches stays as it is. Proven by running the test under the race detector repeatedly through the load gate with the machine loaded, with no such failure, and by a deliberate mismatch on each of the two checks still failing with its reading reported."
---

TestLoadDomainIdiomOwnsTheWholeGroup failed once in the merge queue's race stage on a Linux runner and removed an unrelated pull request from the queue. Two subtests failed the same way: the load script ended with exit status 4 and no stderr. The cause is not established. The script has two exits with status 4 (internal/core/rules/load_test.go): the check that the background job's pid is a number other than 0 or 1 (isPID), and the check straight after it that the job's parent, as ps reports it, is the script's shell (ownChild). The failure left nothing on stderr, so which of the two ended the script is not known. One reading that fits is that on a loaded runner the single reading of ps, taken at once, came back empty or early; nothing recorded shows it. The change under test did not touch the rules package; the pull request passed when it was put back.
