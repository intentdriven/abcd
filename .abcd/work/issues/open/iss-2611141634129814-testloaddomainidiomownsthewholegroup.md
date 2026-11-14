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
remedy: "In the test's ownChild check (internal/core/rules/load_test.go), read the child's parent with a short bounded retry (a few readings some tens of milliseconds apart) before exiting 4, and print what ps returned to stderr on the failing path so a later failure names its cause; the launch text the rule teaches stays as it is. Proven by running the test under the race detector repeatedly through the load gate with the machine loaded, with no exit-4 failure, and by a deliberate mismatch still ending in exit 4 with the reading on stderr."
---

TestLoadDomainIdiomOwnsTheWholeGroup failed once in the merge queue's race stage on a Linux runner and removed an unrelated pull request from the queue. Two subtests failed the same way: the load script ended with exit status 4 and no stderr. Exit 4 is the script's own check, straight after it starts its background job, that the job's parent as ps reports it is the script's shell. On a loaded runner that one reading of ps, taken at once, can come back empty or early, and the script gives up. The change under test did not touch the rules package; the pull request passed when it was put back.
