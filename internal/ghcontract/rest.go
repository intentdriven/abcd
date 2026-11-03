package ghcontract

import (
	_ "embed"
	"strings"
)

// The forge's REST fields the merge gate reads (spc-2610161609350616 step 4):
// who a pull request's author is, the author's and each reviewer's permission
// on the repository, and the pull request's reviews; and those the landing's
// merge-queue watch reads (spc-2609212015054359 step 5): the merge queue's
// workflow runs and a commit's check runs. rest-fields.txt says how they were
// read and how to refresh them.
//
//go:embed rest-fields.txt
var restFields string

var restSet = func() map[string]bool {
	m := map[string]bool{}
	for _, line := range strings.Split(restFields, "\n") {
		if f := strings.TrimSpace(line); f != "" && !strings.HasPrefix(f, "#") {
			m[f] = true
		}
	}
	return m
}()

// RESTField reports whether the forge's REST resource ("collaborator-
// permission", "pull-request-review", "pull-request", "workflow-runs",
// "check-runs") has field, a dotted path
// for a field inside an object ("user.login"), as the pinned list holds it.
func RESTField(resource, field string) bool { return restSet[resource+"."+field] }
