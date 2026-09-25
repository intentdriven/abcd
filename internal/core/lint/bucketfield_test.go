package lint

import (
	"path/filepath"
	"strings"
	"testing"
)

// A store declaring a bucketField is a bucketed store: checkRecordBucketField
// compares the field with the bucket the record is filed under, and a flat
// store declaring one would report "filed under ''" with nothing red
// (iss-2608301634527391).
func TestEveryStoreDeclaringABucketFieldDeclaresBuckets(t *testing.T) {
	for _, s := range recordStores {
		if s.bucketField != "" && !s.bucketed() {
			t.Errorf("store %q declares bucketField %q but no buckets", s.prefix, s.bucketField)
		}
	}
}

// checkRecordBucketField stands down on an absent bucket field and leaves it
// to the required-fields leg, which holds only while the field is in the
// store's required set (iss-2608301808197261 item 1).
func TestEveryBucketFieldIsARequiredProperty(t *testing.T) {
	for _, s := range recordStores {
		if s.bucketField == "" {
			continue
		}
		required := false
		for _, f := range s.requiredFields {
			required = required || f == s.bucketField
		}
		if !required {
			t.Errorf("store %q declares bucketField %q outside its required set, so an absent one is silent", s.prefix, s.bucketField)
		}
	}
}

// An admission that is both cross-bucket and cross-position reports both, so
// the author converges in one lint round (item 2).
func TestAnAdmissionCrossBucketAndCrossPositionReportsBoth(t *testing.T) {
	root := admissionCorpus(t)
	writeFile(t, root, "work/issues/readings/rdg-5/rdi-6.md",
		"---\nschema_version: 1\nid: rdi-6\nrun: rdg-5\nmanifest: sha256:beef\nposition: detection\n"+
			"regime: registrative\npattern: a stated constraint\n---\n\n")
	writeFile(t, root, "work/issues/admissions/rdg-1/adm-2.md",
		"---\nschema_version: 1\nid: adm-2\nrun: rdg-1\nproposal: rdi-6\ngrounds: it widens the frame\n---\n\n")
	fs, err := Lint(admissionSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	rel := filepath.Join("work", "issues", "admissions", "rdg-1", "adm-2.md")
	if !findingWith(fs, rel, ruleRecordSchema, "declares position 'detection'") {
		t.Errorf("the position leg did not report: %+v", fs)
	}
	if !findingWith(fs, rel, ruleRecordSchema, "which is filed under 'rdg-5'") {
		t.Errorf("the bucket leg was skipped behind the position leg: %+v", fs)
	}
}

// A grounds value that is only a zero-width space states nothing, as a blank
// does; strings.TrimSpace does not treat U+200B as whitespace (item 3). An
// alias to an anchor the record does not define carries nothing either (item 4).
func TestAdmissionGroundsOfNothingButAZeroWidthSpaceOrAnAliasIsRefused(t *testing.T) {
	for name, grounds := range map[string]string{
		"zero-width space":        "​",
		"quoted zero-width space": "\"​ \"",
		"undefined alias":         "*a",
	} {
		t.Run(name, func(t *testing.T) {
			root := admissionCorpus(t)
			writeFile(t, root, "work/issues/admissions/rdg-1/adm-3.md",
				"---\nschema_version: 1\nid: adm-3\nrun: rdg-1\nproposal: rdi-2\ngrounds: "+grounds+"\n---\n\n")
			fs, err := Lint(admissionSchemaConfig(), root)
			if err != nil {
				t.Fatal(err)
			}
			if !findingWith(fs, filepath.Join("work", "issues", "admissions", "rdg-1", "adm-3.md"), ruleRecordSchema, "'grounds'") {
				t.Fatalf("grounds %q passed the gate: %+v", strings.ToValidUTF8(grounds, "?"), fs)
			}
		})
	}
}
