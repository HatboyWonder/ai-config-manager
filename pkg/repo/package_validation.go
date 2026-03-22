package repo

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dynatrace-oss/ai-config-manager/v3/pkg/resource"
)

// PackageReferenceIssue describes a single invalid package resource reference.
type PackageReferenceIssue struct {
	Reference  string
	Message    string
	Suggestion string
}

// PackageReferenceIndex stores canonical resource IDs available in a validation context.
type PackageReferenceIndex struct {
	resources map[resource.ResourceType]map[string]struct{}
}

// NewPackageReferenceIndex creates an empty package reference index.
func NewPackageReferenceIndex() *PackageReferenceIndex {
	return &PackageReferenceIndex{
		resources: map[resource.ResourceType]map[string]struct{}{
			resource.Command: {},
			resource.Skill:   {},
			resource.Agent:   {},
		},
	}
}

// Add inserts a canonical resource name into the index.
func (i *PackageReferenceIndex) Add(resType resource.ResourceType, name string) {
	if i == nil || name == "" {
		return
	}

	if _, ok := i.resources[resType]; !ok {
		i.resources[resType] = map[string]struct{}{}
	}

	i.resources[resType][name] = struct{}{}
}

// Exists checks whether a canonical resource name exists in the index.
func (i *PackageReferenceIndex) Exists(resType resource.ResourceType, name string) bool {
	if i == nil {
		return false
	}

	names, ok := i.resources[resType]
	if !ok {
		return false
	}

	_, exists := names[name]
	return exists
}

// CandidateIDs returns sorted canonical IDs for the given type.
func (i *PackageReferenceIndex) CandidateIDs(resType resource.ResourceType) []string {
	if i == nil {
		return nil
	}

	names, ok := i.resources[resType]
	if !ok {
		return nil
	}

	result := make([]string, 0, len(names))
	for name := range names {
		result = append(result, fmt.Sprintf("%s/%s", resType, name))
	}
	sort.Strings(result)
	return result
}

// SuggestCanonicalID suggests a likely intended canonical ID for a missing reference.
func (i *PackageReferenceIndex) SuggestCanonicalID(resType resource.ResourceType, name string) string {
	if i == nil || strings.TrimSpace(name) == "" {
		return ""
	}

	names, ok := i.resources[resType]
	if !ok || len(names) == 0 {
		return ""
	}

	needle := strings.ToLower(name)
	bestName := ""
	bestDistance := -1

	for candidate := range names {
		candidateLower := strings.ToLower(candidate)

		if strings.Contains(candidateLower, needle) || strings.Contains(needle, candidateLower) {
			return fmt.Sprintf("%s/%s", resType, candidate)
		}

		distance := levenshteinDistance(needle, candidateLower)
		if bestDistance == -1 || distance < bestDistance {
			bestDistance = distance
			bestName = candidate
		}
	}

	if bestName == "" {
		return ""
	}

	maxDistance := 3
	if len(needle) > 12 {
		maxDistance = 4
	}
	if bestDistance > maxDistance {
		return ""
	}

	return fmt.Sprintf("%s/%s", resType, bestName)
}

// BuildPackageReferenceIndexFromRoots builds a canonical reference index from one or more source roots.
func BuildPackageReferenceIndexFromRoots(roots []string) (*PackageReferenceIndex, error) {
	index := NewPackageReferenceIndex()

	for _, root := range roots {
		if strings.TrimSpace(root) == "" {
			continue
		}

		absRoot, err := filepath.Abs(root)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve source root %q: %w", root, err)
		}

		manager := NewManagerWithPath(absRoot)
		resources, err := manager.List(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to index source root %q: %w", absRoot, err)
		}

		for _, res := range resources {
			switch res.Type {
			case resource.Command, resource.Skill, resource.Agent:
				index.Add(res.Type, res.Name)
			}
		}
	}

	return index, nil
}

// ValidatePackageReferences validates package resource references against a pre-built context index.
func ValidatePackageReferences(pkg *resource.Package, index *PackageReferenceIndex) []PackageReferenceIssue {
	if pkg == nil {
		return []PackageReferenceIssue{{
			Message: "package cannot be nil",
		}}
	}

	var issues []PackageReferenceIssue
	for _, ref := range pkg.Resources {
		resType, resName, err := resource.ParseResourceReference(ref)
		if err != nil {
			issues = append(issues, PackageReferenceIssue{
				Reference: ref,
				Message:   err.Error(),
			})
			continue
		}

		if index.Exists(resType, resName) {
			continue
		}

		issue := PackageReferenceIssue{
			Reference: ref,
			Message:   fmt.Sprintf("missing referenced resource %q", ref),
		}

		if suggestion := index.SuggestCanonicalID(resType, resName); suggestion != "" {
			issue.Suggestion = fmt.Sprintf("Did you mean %q?", suggestion)
		}

		issues = append(issues, issue)
	}

	return issues
}

func levenshteinDistance(a, b string) int {
	if a == b {
		return 0
	}
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= len(a); i++ {
		curr := make([]int, len(b)+1)
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}

			deletion := prev[j] + 1
			insertion := curr[j-1] + 1
			substitution := prev[j-1] + cost

			curr[j] = minInt(deletion, insertion, substitution)
		}
		prev = curr
	}

	return prev[len(b)]
}

func minInt(values ...int) int {
	best := values[0]
	for _, v := range values[1:] {
		if v < best {
			best = v
		}
	}
	return best
}
