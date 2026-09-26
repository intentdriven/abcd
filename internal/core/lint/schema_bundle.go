package lint

// The record_schema family's bundle leg (itd-34, spc-2609211859391533 scope 5).
//
// A bundle exists only in its members' frontmatter: each carries
// `kind: bundle-member` and `bundle: <name>`, and nothing else declares the
// bundle. So a member naming a bundle no other record names is a bundle of one
// nobody planned — a mistyped name, or a mate edited away by hand — and the one
// legitimate bundle of one is the survivor a supersession leaves, whose
// `reclassification_history` says the bundle now has one member (decision 3;
// `abcd intent reclassify` writes that line). This leg refuses the rest,
// naming the record.
//
// It reads a bundle only where the record names one. A bundle-member carrying
// no `bundle:` at all is a convention the brief states and no verb produces
// (both writers stamp the name); the shipped records that predate the verbs and
// carry none are shipped, so their kind is settled (decision 4) and a finding
// on them would be one nobody can act on.
//
// The kind-versus-shelf half of the same criterion is intent_lifecycle's: it
// already holds the kind every intent bucket admits, and a second finding here
// for the same line would be one defect reported twice.

import (
	"strings"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/intentbundle"
)

// checkIntentBundles runs the bundle leg over the scanned records.
func checkIntentBundles(records []schemaRecord, severity string) []Finding {
	named := map[string][]schemaRecord{}
	var members []schemaRecord
	for _, r := range records {
		if r.store.prefix != "itd" {
			continue
		}
		name := recordBundle(r)
		if name == "" {
			continue
		}
		named[name] = append(named[name], r)
		if kind, _ := frontmatter.ScalarString(r.fields["kind"].value); kind == "bundle-member" {
			members = append(members, r)
		}
	}
	var out []Finding
	for _, r := range members {
		name := recordBundle(r)
		if len(named[name]) > 1 {
			continue
		}
		history := r.fields["reclassification_history"].value
		if b, ok := r.blocks["reclassification_history"]; ok {
			history += " " + b
		}
		if strings.Contains(history, intentbundle.OneMember(name)) {
			continue
		}
		line := r.fields["bundle"].line
		if line == 0 {
			line = 1
		}
		out = append(out, Finding{
			File: r.rel, Line: line, RuleID: ruleRecordSchema, Severity: severity,
			Message: "bundle-member names bundle '" + name + "', which no other record names and whose history does not say it now has one member; " +
				"a bundle is its members' shared name, so name a bundle its mates carry, or retire the member with `abcd intent reclassify`",
		})
	}
	return out
}

// recordBundle is the bundle a record names, or "" when it names none.
func recordBundle(r schemaRecord) string {
	name, ok := frontmatter.ScalarString(r.fields["bundle"].value)
	if !ok || frontmatter.IsNull(name) {
		return ""
	}
	return name
}
