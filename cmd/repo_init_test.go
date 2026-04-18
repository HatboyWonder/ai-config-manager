package cmd

import (
	"strings"
	"testing"
)

func TestRepoInitHelpClarifiesApplyRelationship(t *testing.T) {
	help := repoInitCmd.Long
	for _, expected := range []string{
		"repo init is local bootstrap only",
		"repo apply-manifest <path-or-url>",
		"repo show-manifest",
		"bootstrapping from a shared manifest",
	} {
		if !strings.Contains(help, expected) {
			t.Fatalf("expected repo init help to mention %q", expected)
		}
	}
}
