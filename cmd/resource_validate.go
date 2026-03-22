package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dynatrace-oss/ai-config-manager/v3/pkg/output"
	"github.com/dynatrace-oss/ai-config-manager/v3/pkg/repo"
	"github.com/dynatrace-oss/ai-config-manager/v3/pkg/repomanifest"
	"github.com/dynatrace-oss/ai-config-manager/v3/pkg/resource"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const (
	resourceValidateExitValidationError = 1
	resourceValidateExitUsageError      = 2
)

type validateDiagnostic struct {
	Severity         string `json:"severity" yaml:"severity"`
	Code             string `json:"code" yaml:"code"`
	Message          string `json:"message" yaml:"message"`
	Field            string `json:"field,omitempty" yaml:"field,omitempty"`
	FilePath         string `json:"file_path,omitempty" yaml:"file_path,omitempty"`
	ResourceName     string `json:"resource_name,omitempty" yaml:"resource_name,omitempty"`
	ResourceType     string `json:"resource_type,omitempty" yaml:"resource_type,omitempty"`
	Suggestion       string `json:"suggestion,omitempty" yaml:"suggestion,omitempty"`
	MissingReference string `json:"missing_reference,omitempty" yaml:"missing_reference,omitempty"`
}

type validateSummary struct {
	ErrorCount   int `json:"error_count" yaml:"error_count"`
	WarningCount int `json:"warning_count" yaml:"warning_count"`
}

type validateContext struct {
	Kind         string `json:"kind" yaml:"kind"`
	SourceRoot   string `json:"source_root,omitempty" yaml:"source_root,omitempty"`
	RepoManifest string `json:"repo_manifest,omitempty" yaml:"repo_manifest,omitempty"`
}

type resourceValidationResult struct {
	Target       string               `json:"target" yaml:"target"`
	ResolvedPath string               `json:"resolved_path,omitempty" yaml:"resolved_path,omitempty"`
	ResolvedID   string               `json:"resolved_id,omitempty" yaml:"resolved_id,omitempty"`
	ResourceType string               `json:"resource_type,omitempty" yaml:"resource_type,omitempty"`
	Mode         string               `json:"mode" yaml:"mode"`
	Context      validateContext      `json:"context" yaml:"context"`
	Valid        bool                 `json:"valid" yaml:"valid"`
	Diagnostics  []validateDiagnostic `json:"diagnostics,omitempty" yaml:"diagnostics,omitempty"`
	Summary      validateSummary      `json:"summary" yaml:"summary"`
}

type validateCanonicalTarget struct {
	Type resource.ResourceType
	Name string
	Raw  string
}

type packageValidationContext struct {
	kind         string
	roots        []string
	repoManifest string
	sourceRoot   string
}

type resourceValidateRunResult struct {
	Output   resourceValidationResult
	ExitCode int
}

var (
	resourceValidateFormatFlag       string
	resourceValidateSourceRootFlag   string
	resourceValidateRepoManifestFlag string
)

var resourceValidateCmd = &cobra.Command{
	Use:   "validate <resource-id-or-path>",
	Short: "Validate a resource by canonical ID or path",
	Long: `Validate a single resource by filesystem path or canonical resource ID.

Resolution rules:
  1. If the target exists on disk, it is treated as a path.
  2. Otherwise it must parse as a canonical ID (type/name).

For canonical IDs, resolution context precedence is:
  --source-root, local repo (if available), then --repo-manifest.

Exit status:
  0 - Validation passed
  1 - Validation failed
  2 - Usage/context/setup error`,
	Args:               cobra.ExactArgs(1),
	ValidArgsFunction:  completeResourceValidateArgs,
	DisableFlagParsing: false,
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := output.ParseFormat(resourceValidateFormatFlag); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(resourceValidateExitUsageError)
		}

		result := runResourceValidate(args[0], resourceValidateOptions{
			format:       resourceValidateFormatFlag,
			sourceRoot:   resourceValidateSourceRootFlag,
			repoManifest: resourceValidateRepoManifestFlag,
		})

		if err := outputResourceValidateResult(&result.Output, result.Output.Valid, result.ExitCode, resourceValidateFormatFlag); err != nil {
			return err
		}

		if result.ExitCode != 0 {
			os.Exit(result.ExitCode)
		}

		return nil
	},
}

type resourceValidateOptions struct {
	format       string
	sourceRoot   string
	repoManifest string
}

func runResourceValidate(target string, opts resourceValidateOptions) resourceValidateRunResult {
	resolvedSourceRoot, err := normalizeSourceRoot(opts.sourceRoot)
	if err != nil {
		res := buildSetupErrorResult(target, "invalid_source_root", err)
		if opts.sourceRoot != "" {
			res.Context = validateContext{Kind: "source-root", SourceRoot: opts.sourceRoot}
		}
		return resourceValidateRunResult{Output: res, ExitCode: resourceValidateExitUsageError}
	}

	resolvedPath, pathExists, err := pathIfExists(target)
	if err != nil {
		res := buildSetupErrorResult(target, "path_resolution_error", err)
		return resourceValidateRunResult{Output: res, ExitCode: resourceValidateExitUsageError}
	}

	if pathExists {
		res := validatePathTarget(target, resolvedPath, resourceValidateOptions{sourceRoot: resolvedSourceRoot, repoManifest: opts.repoManifest})
		if res.Valid {
			return resourceValidateRunResult{Output: res, ExitCode: 0}
		}
		return resourceValidateRunResult{Output: res, ExitCode: resourceValidateExitValidationError}
	}

	canonical, canonicalErr := parseCanonicalValidateTarget(target)
	if canonicalErr != nil {
		res := buildSetupErrorResult(target, "invalid_target", fmt.Errorf("target must be an existing path or canonical ID (type/name): %w", canonicalErr))
		return resourceValidateRunResult{Output: res, ExitCode: resourceValidateExitUsageError}
	}

	resolvedCanonicalPath, ctxInfo, resolveErr := resolveCanonicalTargetPath(canonical, resolvedSourceRoot, opts.repoManifest)
	if resolveErr != nil {
		res := buildSetupErrorResult(target, "context_required", resolveErr)
		res.ResolvedID = canonical.Raw
		res.ResourceType = string(canonical.Type)
		res.Context = ctxInfo
		return resourceValidateRunResult{Output: res, ExitCode: resourceValidateExitUsageError}
	}

	res := validatePathTarget(target, resolvedCanonicalPath, resourceValidateOptions{sourceRoot: resolvedSourceRoot, repoManifest: opts.repoManifest})
	res.ResolvedID = canonical.Raw
	res.Context = ctxInfo
	if res.ResourceType == "" {
		res.ResourceType = string(canonical.Type)
	}

	if res.Valid {
		return resourceValidateRunResult{Output: res, ExitCode: 0}
	}
	return resourceValidateRunResult{Output: res, ExitCode: resourceValidateExitValidationError}
}

func normalizeSourceRoot(sourceRoot string) (string, error) {
	if strings.TrimSpace(sourceRoot) == "" {
		return "", nil
	}

	abs, err := filepath.Abs(sourceRoot)
	if err != nil {
		return "", fmt.Errorf("failed to resolve --source-root: %w", err)
	}

	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("failed to stat --source-root %q: %w", abs, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("--source-root must be a directory: %s", abs)
	}

	return abs, nil
}

func pathIfExists(target string) (string, bool, error) {
	if strings.TrimSpace(target) == "" {
		return "", false, fmt.Errorf("target cannot be empty")
	}

	abs, err := filepath.Abs(target)
	if err != nil {
		return "", false, fmt.Errorf("failed to resolve target path: %w", err)
	}

	_, err = os.Stat(abs)
	if err == nil {
		return abs, true, nil
	}
	if os.IsNotExist(err) {
		return abs, false, nil
	}

	return "", false, fmt.Errorf("failed to stat target path: %w", err)
}

func parseCanonicalValidateTarget(arg string) (validateCanonicalTarget, error) {
	parts := strings.SplitN(arg, "/", 2)
	if len(parts) != 2 {
		return validateCanonicalTarget{}, fmt.Errorf("invalid format: must be 'type/name' (e.g., skill/my-skill, command/my-command, agent/my-agent, package/my-package)")
	}

	typeStr := strings.TrimSpace(parts[0])
	name := strings.TrimSpace(parts[1])
	if name == "" {
		return validateCanonicalTarget{}, fmt.Errorf("resource name cannot be empty")
	}

	var resType resource.ResourceType
	switch strings.ToLower(typeStr) {
	case "skill", "skills":
		resType = resource.Skill
	case "command", "commands":
		resType = resource.Command
	case "agent", "agents":
		resType = resource.Agent
	case "package", "packages":
		resType = resource.PackageType
	default:
		return validateCanonicalTarget{}, fmt.Errorf("invalid resource type '%s': must be one of 'skill', 'command', 'agent', or 'package'", typeStr)
	}

	return validateCanonicalTarget{
		Type: resType,
		Name: name,
		Raw:  fmt.Sprintf("%s/%s", string(resType), name),
	}, nil
}

func resolveCanonicalTargetPath(target validateCanonicalTarget, sourceRoot string, repoManifestInput string) (string, validateContext, error) {
	if sourceRoot != "" {
		resolvedPath := canonicalPathFromRoot(sourceRoot, target)
		return resolvedPath, validateContext{Kind: "source-root", SourceRoot: sourceRoot}, nil
	}

	if repoRoot, ok := detectLocalRepoRoot(); ok {
		resolvedPath := canonicalPathFromRoot(repoRoot, target)
		return resolvedPath, validateContext{Kind: "repo", SourceRoot: repoRoot}, nil
	}

	if strings.TrimSpace(repoManifestInput) != "" {
		resolvedPath, err := resolveFromRepoManifest(target, repoManifestInput)
		ctx := validateContext{Kind: "repo-manifest", RepoManifest: repoManifestInput}
		if err != nil {
			return "", ctx, err
		}
		return resolvedPath, ctx, nil
	}

	return "", validateContext{Kind: "none"}, fmt.Errorf("canonical ID %q requires context: provide --source-root or --repo-manifest", target.Raw)
}

func detectLocalRepoRoot() (string, bool) {
	repoPath := repo.ResolveRepoPath()
	if strings.TrimSpace(repoPath) == "" {
		return "", false
	}

	info, err := os.Stat(repoPath)
	if err != nil || !info.IsDir() {
		return "", false
	}

	manifestPath := filepath.Join(repoPath, repomanifest.ManifestFileName)
	if _, err := os.Stat(manifestPath); err != nil {
		return "", false
	}

	abs, err := filepath.Abs(repoPath)
	if err != nil {
		return repoPath, true
	}

	return abs, true
}

func resolveFromRepoManifest(target validateCanonicalTarget, manifestInput string) (string, error) {
	manifest, err := repomanifest.LoadForApply(manifestInput)
	if err != nil {
		return "", fmt.Errorf("failed to load --repo-manifest: %w", err)
	}

	for _, src := range manifest.Sources {
		if src == nil || src.Path == "" {
			continue
		}

		resolvedRoot, err := filepath.Abs(src.Path)
		if err != nil {
			continue
		}

		candidate := canonicalPathFromRoot(resolvedRoot, target)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("resource %q not found in any local path source from manifest", target.Raw)
}

func canonicalPathFromRoot(root string, target validateCanonicalTarget) string {
	switch target.Type {
	case resource.Command:
		return filepath.Join(root, "commands", target.Name+".md")
	case resource.Skill:
		return filepath.Join(root, "skills", target.Name)
	case resource.Agent:
		return filepath.Join(root, "agents", target.Name+".md")
	case resource.PackageType:
		return filepath.Join(root, "packages", target.Name+".package.json")
	default:
		return ""
	}
}

func validatePathTarget(originalTarget, resolvedPath string, opts resourceValidateOptions) resourceValidationResult {
	result := resourceValidationResult{
		Target:       originalTarget,
		ResolvedPath: resolvedPath,
		Mode:         "static",
		Context:      validateContext{Kind: "none"},
		Valid:        false,
	}

	if strings.HasSuffix(strings.ToLower(resolvedPath), ".package.json") {
		return validatePackagePathTarget(originalTarget, resolvedPath, opts)
	}

	res, loadErr := resource.Load(resolvedPath)
	if loadErr == nil {
		result.ResourceType = string(res.Type)
		switch res.Type {
		case resource.Skill:
			if err := resource.ValidateSkill(resolvedPath); err != nil {
				result.Diagnostics = []validateDiagnostic{diagnosticFromError("validation_error", err)}
			} else {
				result.Valid = true
			}
		case resource.Agent:
			if err := resource.ValidateAgent(resolvedPath); err != nil {
				result.Diagnostics = []validateDiagnostic{diagnosticFromError("validation_error", err)}
			} else {
				result.Valid = true
			}
		case resource.Command:
			if err := validateCommandStaticStandalone(resolvedPath); err != nil {
				result.Diagnostics = []validateDiagnostic{diagnosticFromError("validation_error", err)}
			} else {
				result.Valid = true
			}
		default:
			result.Diagnostics = []validateDiagnostic{{
				Severity: "error",
				Code:     "unsupported_resource_type",
				Message:  fmt.Sprintf("resource type %q is not supported by static validation", res.Type),
			}}
		}
	} else {
		// Fallback for standalone command files outside commands/ directories.
		if strings.EqualFold(filepath.Ext(resolvedPath), ".md") && strings.Contains(loadErr.Error(), "command file must be in a 'commands/' directory") {
			if cmdErr := validateCommandStaticStandalone(resolvedPath); cmdErr == nil {
				result.ResourceType = string(resource.Command)
				result.Valid = true
			} else {
				result.Diagnostics = []validateDiagnostic{diagnosticFromError("validation_error", cmdErr)}
				if result.ResourceType == "" {
					result.ResourceType = string(resource.Command)
				}
			}
		} else {
			result.Diagnostics = []validateDiagnostic{diagnosticFromError("validation_error", loadErr)}
		}
	}

	result.Summary.ErrorCount = len(result.Diagnostics)
	result.Summary.WarningCount = 0
	return result
}

func validatePackagePathTarget(originalTarget, resolvedPath string, opts resourceValidateOptions) resourceValidationResult {
	result := resourceValidationResult{
		Target:       originalTarget,
		ResolvedPath: resolvedPath,
		ResourceType: string(resource.PackageType),
		Mode:         "static+contextual",
		Context:      validateContext{Kind: "none"},
		Valid:        false,
	}

	pkg, err := resource.LoadPackage(resolvedPath)
	if err != nil {
		result.Mode = "static"
		result.Diagnostics = []validateDiagnostic{diagnosticFromError("resource_validation_error", err)}
		result.Summary = validateSummary{ErrorCount: len(result.Diagnostics), WarningCount: 0}
		return result
	}

	ctx, err := buildPackageValidationContext(resolvedPath, opts.sourceRoot, opts.repoManifest)
	if err != nil {
		result.Diagnostics = []validateDiagnostic{{
			Severity: "error",
			Code:     "context_required",
			Message:  err.Error(),
		}}
		result.Summary = validateSummary{ErrorCount: len(result.Diagnostics), WarningCount: 0}
		return result
	}

	result.Context = validateContext{Kind: ctx.kind, SourceRoot: ctx.sourceRoot, RepoManifest: ctx.repoManifest}

	index, err := repo.BuildPackageReferenceIndexFromRoots(ctx.roots)
	if err != nil {
		result.Diagnostics = []validateDiagnostic{{
			Severity: "error",
			Code:     "context_build_failed",
			Message:  err.Error(),
		}}
		result.Summary = validateSummary{ErrorCount: len(result.Diagnostics), WarningCount: 0}
		return result
	}

	issues := repo.ValidatePackageReferences(pkg, index)
	result.Diagnostics = diagnosticsFromPackageIssues(issues, index)
	result.Valid = len(result.Diagnostics) == 0
	result.Summary = validateSummary{ErrorCount: len(result.Diagnostics), WarningCount: 0}
	return result
}

func buildPackageValidationContext(packagePath, sourceRoot, repoManifestInput string) (packageValidationContext, error) {
	if strings.TrimSpace(sourceRoot) != "" {
		absRoot, err := filepath.Abs(sourceRoot)
		if err != nil {
			return packageValidationContext{}, fmt.Errorf("failed to resolve --source-root: %w", err)
		}
		return packageValidationContext{
			kind:       "source-root",
			roots:      []string{absRoot},
			sourceRoot: absRoot,
		}, nil
	}

	if repoRoot, ok := detectLocalRepoRoot(); ok {
		return packageValidationContext{
			kind:       "repo",
			roots:      []string{repoRoot},
			sourceRoot: repoRoot,
		}, nil
	}

	if strings.TrimSpace(repoManifestInput) != "" {
		manifest, err := repomanifest.LoadForApply(repoManifestInput)
		if err != nil {
			return packageValidationContext{}, fmt.Errorf("failed to load --repo-manifest: %w", err)
		}

		roots := collectManifestSourceRoots(manifest)
		if len(roots) == 0 {
			return packageValidationContext{}, fmt.Errorf("--repo-manifest must include at least one local path source")
		}

		return packageValidationContext{
			kind:         "repo-manifest",
			roots:        roots,
			repoManifest: repoManifestInput,
		}, nil
	}

	inferredRoot, ok := inferSourceRootFromPackagePath(packagePath)
	if ok {
		return packageValidationContext{
			kind:       "source-root",
			roots:      []string{inferredRoot},
			sourceRoot: inferredRoot,
		}, nil
	}

	return packageValidationContext{}, fmt.Errorf("package reference validation requires context: provide --source-root or --repo-manifest")
}

func collectManifestSourceRoots(manifest *repomanifest.Manifest) []string {
	if manifest == nil {
		return nil
	}

	seen := make(map[string]struct{})
	var roots []string
	for _, src := range manifest.Sources {
		if src == nil || strings.TrimSpace(src.Path) == "" {
			continue
		}

		absPath, err := filepath.Abs(src.Path)
		if err != nil {
			continue
		}

		if _, ok := seen[absPath]; ok {
			continue
		}
		seen[absPath] = struct{}{}
		roots = append(roots, absPath)
	}

	sort.Strings(roots)
	return roots
}

func inferSourceRootFromPackagePath(packagePath string) (string, bool) {
	absPackagePath, err := filepath.Abs(packagePath)
	if err != nil {
		return "", false
	}

	parent := filepath.Dir(absPackagePath)
	if filepath.Base(parent) != "packages" {
		return "", false
	}

	root := filepath.Dir(parent)
	if hasResourceLayout(root) {
		return root, true
	}

	return "", false
}

func hasResourceLayout(root string) bool {
	required := []string{"commands", "skills", "agents", "packages"}
	for _, dir := range required {
		info, err := os.Stat(filepath.Join(root, dir))
		if err != nil || !info.IsDir() {
			return false
		}
	}
	return true
}

func diagnosticsFromPackageIssues(issues []repo.PackageReferenceIssue, index *repo.PackageReferenceIndex) []validateDiagnostic {
	if len(issues) == 0 {
		return nil
	}

	diagnostics := make([]validateDiagnostic, 0, len(issues))
	for _, issue := range issues {
		d := validateDiagnostic{
			Severity:         "error",
			Code:             "missing_package_ref",
			Message:          issue.Message,
			Suggestion:       issue.Suggestion,
			MissingReference: issue.Reference,
			ResourceType:     string(resource.PackageType),
		}

		if issue.Reference == "" {
			d.Code = "invalid_package_ref"
		} else {
			resType, _, err := resource.ParseResourceReference(issue.Reference)
			if err == nil && d.Suggestion == "" {
				candidates := index.CandidateIDs(resType)
				if len(candidates) > 0 {
					limit := 3
					if len(candidates) < limit {
						limit = len(candidates)
					}
					d.Suggestion = fmt.Sprintf("Available %s IDs include: %s", resType, strings.Join(candidates[:limit], ", "))
				}
			}
		}

		diagnostics = append(diagnostics, d)
	}

	return diagnostics
}

func validateCommandStaticStandalone(filePath string) error {
	if err := resource.ValidateCommand(filePath); err == nil {
		return nil
	}

	_, err := resource.LoadCommandWithBase(filePath, filepath.Dir(filePath))
	return err
}

func diagnosticFromError(defaultCode string, err error) validateDiagnostic {
	d := validateDiagnostic{
		Severity: "error",
		Code:     defaultCode,
		Message:  err.Error(),
	}

	var validationErr *resource.ValidationError
	if errors.As(err, &validationErr) {
		d.Code = "resource_validation_error"
		d.FilePath = validationErr.FilePath
		d.ResourceName = validationErr.ResourceName
		d.ResourceType = validationErr.ResourceType
		d.Field = validationErr.FieldName
		d.Suggestion = validationErr.Suggestion
	}

	return d
}

func buildSetupErrorResult(target, code string, err error) resourceValidationResult {
	return resourceValidationResult{
		Target: target,
		Mode:   "static",
		Context: validateContext{
			Kind: "none",
		},
		Valid: false,
		Diagnostics: []validateDiagnostic{{
			Severity: "error",
			Code:     code,
			Message:  err.Error(),
		}},
		Summary: validateSummary{ErrorCount: 1, WarningCount: 0},
	}
}

func outputResourceValidateResult(result *resourceValidationResult, valid bool, exitCode int, format string) error {
	parsedFormat, err := output.ParseFormat(format)
	if err != nil {
		return err
	}

	result.Valid = valid
	if result.Summary.ErrorCount == 0 {
		result.Summary.ErrorCount = len(result.Diagnostics)
	}

	switch parsedFormat {
	case output.JSON:
		return output.EncodeJSON(os.Stdout, result)
	case output.YAML:
		encoder := yaml.NewEncoder(os.Stdout)
		defer func() { _ = encoder.Close() }()
		return encoder.Encode(result)
	case output.Table:
		displayResourceValidationTable(result, exitCode)
		return nil
	default:
		return fmt.Errorf("unsupported format: %s", parsedFormat)
	}
}

func displayResourceValidationTable(result *resourceValidationResult, exitCode int) {
	status := statusIconOK
	statusText := "valid"
	if !result.Valid {
		status = statusIconFail
		statusText = "invalid"
	}

	fmt.Printf("%s Resource validation: %s\n\n", status, statusText)

	table := output.NewTable("Field", "Value")
	table.WithResponsive().WithDynamicColumn(1).WithMinColumnWidths(16, 40)
	table.AddRow("Target", result.Target)
	if result.ResolvedID != "" {
		table.AddRow("Resolved ID", result.ResolvedID)
	}
	if result.ResolvedPath != "" {
		table.AddRow("Resolved Path", result.ResolvedPath)
	}
	if result.ResourceType != "" {
		table.AddRow("Resource Type", result.ResourceType)
	}
	table.AddRow("Mode", result.Mode)
	table.AddRow("Context", result.Context.Kind)
	if result.Context.SourceRoot != "" {
		table.AddRow("Source Root", result.Context.SourceRoot)
	}
	if result.Context.RepoManifest != "" {
		table.AddRow("Repo Manifest", result.Context.RepoManifest)
	}
	table.AddRow("Errors", fmt.Sprintf("%d", result.Summary.ErrorCount))
	table.AddRow("Warnings", fmt.Sprintf("%d", result.Summary.WarningCount))
	_ = table.Format(output.Table)

	if len(result.Diagnostics) > 0 {
		fmt.Println()
		diagTable := output.NewTable("Severity", "Code", "Message")
		diagTable.WithResponsive().WithDynamicColumn(2).WithMinColumnWidths(8, 24, 40)
		for _, diag := range result.Diagnostics {
			diagTable.AddRow(diag.Severity, diag.Code, diag.Message)
		}
		_ = diagTable.Format(output.Table)
	}

	if exitCode == resourceValidateExitUsageError {
		fmt.Println()
		fmt.Println("Status: usage/context/setup error (exit code 2)")
	}
}

func completeResourceValidateArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return completeResourcesWithOptions(completionOptions{
		includePackages: true,
		multiArg:        false,
	})(cmd, args, toComplete)
}

func init() {
	resourceCmd.AddCommand(resourceValidateCmd)

	resourceValidateCmd.Flags().StringVar(&resourceValidateFormatFlag, "format", "table", "Output format (table|json|yaml)")
	_ = resourceValidateCmd.RegisterFlagCompletionFunc("format", completeFormatFlag)

	resourceValidateCmd.Flags().StringVar(&resourceValidateSourceRootFlag, "source-root", "", "Source root for canonical resource ID resolution")
	resourceValidateCmd.Flags().StringVar(&resourceValidateRepoManifestFlag, "repo-manifest", "", "Manifest path/url for canonical resource ID resolution")
}
