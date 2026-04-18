package cmd

import "testing"

func withRepoAddFlagsReset(t *testing.T, fn func()) {
	t.Helper()

	originalForce := forceFlag
	originalSkip := skipExistingFlag
	originalDryRun := dryRunFlag
	originalFilters := append([]string(nil), filterFlags...)
	originalFormat := addFormatFlag
	originalName := nameFlag
	originalDiscovery := discoveryFlag
	originalRef := refFlag
	originalSubpath := subpathFlag
	originalSilent := syncSilentMode

	forceFlag = false
	skipExistingFlag = false
	dryRunFlag = false
	filterFlags = nil
	addFormatFlag = "table"
	nameFlag = ""
	discoveryFlag = "auto"
	refFlag = ""
	subpathFlag = ""
	syncSilentMode = false

	defer func() {
		forceFlag = originalForce
		skipExistingFlag = originalSkip
		dryRunFlag = originalDryRun
		filterFlags = originalFilters
		addFormatFlag = originalFormat
		nameFlag = originalName
		discoveryFlag = originalDiscovery
		refFlag = originalRef
		subpathFlag = originalSubpath
		syncSilentMode = originalSilent
	}()

	fn()
}
