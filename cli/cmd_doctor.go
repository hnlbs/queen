package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/honeynil/queen"
	"github.com/spf13/cobra"
)

const (
	statusPass    = "pass"
	statusWarning = "warning"
	statusFail    = "fail"
)

// DoctorResult represents a health check result.
type DoctorResult struct {
	Check    string `json:"check"`
	Status   string `json:"status"` // "statusPass", "statusWarning", "statusFail"
	Message  string `json:"message"`
	Details  string `json:"details,omitempty"`
	Severity string `json:"severity,omitempty"`
}

func (app *App) doctorCmd() *cobra.Command {
	var (
		deep bool
		gaps bool
		fix  bool
	)

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Run migration health checks and diagnostics",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			q, err := app.setupQueen(ctx)
			if err != nil {
				return err
			}
			defer func() { _ = q.Close() }()

			results := make([]DoctorResult, 0)

			// Run health checks
			results = append(results, checkDatabaseConnection(ctx, q))
			results = append(results, checkMigrationTable(ctx, q))
			results = append(results, checkChecksums(ctx, q))

			if gaps || !deep {
				results = append(results, checkGaps(ctx, q))
			}

			results = append(results, checkRegistrationSync(ctx, q))

			if deep {
				results = append(results, checkSchemaConsistency(ctx, q))
			}

			// Auto-fix if requested
			if fix {
				results = append(results, attemptAutoFix(ctx, q, results)...)
			}

			// Output results
			if app.config.JSON {
				return outputDoctorJSON(results)
			}

			return outputDoctorTable(results)
		},
	}

	cmd.Flags().BoolVar(&deep, "deep", false, "Deep schema validation (slower)")
	cmd.Flags().BoolVar(&gaps, "gaps", false, "Only check for gaps")
	cmd.Flags().BoolVar(&fix, "fix", false, "Attempt to auto-fix issues")

	return cmd
}

// checkDatabaseConnection verifies database connectivity.
func checkDatabaseConnection(ctx context.Context, q *queen.Queen) DoctorResult {
	// Try to load applied migrations as a connectivity test
	_, err := q.Driver().GetApplied(ctx)
	if err != nil {
		return DoctorResult{
			Check:   "Database Connection",
			Status:  statusFail,
			Message: "Failed to connect to database",
			Details: err.Error(),
		}
	}

	return DoctorResult{
		Check:   "Database Connection",
		Status:  statusPass,
		Message: "Database is accessible",
	}
}

// checkMigrationTable verifies the migration table exists and is accessible.
func checkMigrationTable(ctx context.Context, q *queen.Queen) DoctorResult {
	applied, err := q.Driver().GetApplied(ctx)
	if err != nil {
		return DoctorResult{
			Check:   "Migration Table",
			Status:  statusFail,
			Message: "Migration table is not accessible",
			Details: err.Error(),
		}
	}

	return DoctorResult{
		Check:   "Migration Table",
		Status:  statusPass,
		Message: fmt.Sprintf("Migration table exists with %d records", len(applied)),
	}
}

// checkChecksums validates that applied migrations haven't been modified.
func checkChecksums(ctx context.Context, q *queen.Queen) DoctorResult {
	statuses, err := q.Status(ctx)
	if err != nil {
		return DoctorResult{
			Check:   "Checksum Validation",
			Status:  statusFail,
			Message: "Failed to validate checksums",
			Details: err.Error(),
		}
	}

	modified := make([]string, 0)
	for _, s := range statuses {
		if s.Status == queen.StatusModified {
			modified = append(modified, s.Version)
		}
	}

	if len(modified) > 0 {
		return DoctorResult{
			Check:    "Checksum Validation",
			Status:   statusFail,
			Message:  fmt.Sprintf("%d migration(s) have been modified after being applied", len(modified)),
			Details:  fmt.Sprintf("Modified versions: %s", strings.Join(modified, ", ")),
			Severity: "error",
		}
	}

	return DoctorResult{
		Check:   "Checksum Validation",
		Status:  statusPass,
		Message: "All applied migrations match their checksums",
	}
}

// checkGaps checks for migration gaps.
func checkGaps(ctx context.Context, q *queen.Queen) DoctorResult {
	gaps, err := q.DetectGaps(ctx)
	if err != nil {
		return DoctorResult{
			Check:   "Gap Detection",
			Status:  statusFail,
			Message: "Failed to detect gaps",
			Details: err.Error(),
		}
	}

	if len(gaps) == 0 {
		return DoctorResult{
			Check:   "Gap Detection",
			Status:  statusPass,
			Message: "No gaps detected",
		}
	}

	// Count by severity
	errors := 0
	statusWarnings := 0
	for _, gap := range gaps {
		if gap.Severity == "error" {
			errors++
		} else {
			statusWarnings++
		}
	}

	status := statusWarning
	if errors > 0 {
		status = statusFail
	}

	return DoctorResult{
		Check:   "Gap Detection",
		Status:  status,
		Message: fmt.Sprintf("Found %d gap(s): %d errors, %d statusWarnings", len(gaps), errors, statusWarnings),
		Details: "Run 'queen gap detect' for details",
	}
}

// checkRegistrationSync checks if code and database are in sync.
func checkRegistrationSync(ctx context.Context, q *queen.Queen) DoctorResult {
	statuses, err := q.Status(ctx)
	if err != nil {
		return DoctorResult{
			Check:   "Registration Sync",
			Status:  statusFail,
			Message: "Failed to check registration sync",
			Details: err.Error(),
		}
	}

	applied, err := q.Driver().GetApplied(ctx)
	if err != nil {
		return DoctorResult{
			Check:   "Registration Sync",
			Status:  statusFail,
			Message: "Failed to get applied migrations",
			Details: err.Error(),
		}
	}

	// Build map of registered versions
	registered := make(map[string]bool)
	for _, s := range statuses {
		registered[s.Version] = true
	}

	// Find unregistered migrations
	unregistered := make([]string, 0)
	for _, a := range applied {
		if !registered[a.Version] {
			unregistered = append(unregistered, a.Version)
		}
	}

	if len(unregistered) > 0 {
		return DoctorResult{
			Check:    "Registration Sync",
			Status:   statusFail,
			Message:  fmt.Sprintf("%d applied migration(s) are not registered in code", len(unregistered)),
			Details:  fmt.Sprintf("Unregistered: %s", strings.Join(unregistered, ", ")),
			Severity: "error",
		}
	}

	return DoctorResult{
		Check:   "Registration Sync",
		Status:  statusPass,
		Message: "All applied migrations are registered in code",
	}
}

// checkSchemaConsistency performs deep schema validation.
func checkSchemaConsistency(_ context.Context, _ *queen.Queen) DoctorResult {
	// This is a placeholder for future schema validation
	// Could check for:
	// - Orphaned tables/columns
	// - Missing indexes referenced in code
	// - Foreign key integrity
	// etc.

	return DoctorResult{
		Check:   "Schema Consistency",
		Status:  statusPass,
		Message: "Deep schema validation not yet implemented",
		Details: "This feature is planned for a future release",
	}
}

// attemptAutoFix tries to automatically fix detected issues.
func attemptAutoFix(_ context.Context, _ *queen.Queen, results []DoctorResult) []DoctorResult {
	fixes := make([]DoctorResult, 0)

	// Check what can be fixed
	for _, result := range results {
		if result.Status != statusFail {
			continue
		}

		switch result.Check {
		case "Gap Detection":
			// Could potentially fill gaps automatically
			fixes = append(fixes, DoctorResult{
				Check:   "Auto-Fix: Gaps",
				Status:  statusWarning,
				Message: "Gap auto-fix not yet implemented",
				Details: "Use 'queen gap fill' to manually fix gaps",
			})

		case "Checksum Validation":
			fixes = append(fixes, DoctorResult{
				Check:   "Auto-Fix: Checksums",
				Status:  statusWarning,
				Message: "Checksum mismatches cannot be auto-fixed",
				Details: "This indicates migrations were modified after being applied",
			})

		case "Registration Sync":
			fixes = append(fixes, DoctorResult{
				Check:   "Auto-Fix: Registration",
				Status:  statusWarning,
				Message: "Registration sync cannot be auto-fixed",
				Details: "Add missing migrations to your code",
			})
		}
	}

	if len(fixes) == 0 {
		fixes = append(fixes, DoctorResult{
			Check:   "Auto-Fix",
			Status:  statusPass,
			Message: "No auto-fixable issues found",
		})
	}

	return fixes
}

// outputDoctorTable prints doctor results in a formatted table.
func outputDoctorTable(results []DoctorResult) error {
	fmt.Println("Queen Migration Health Check")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println()

	statusPassed := 0
	statusWarnings := 0
	statusFailed := 0

	for _, result := range results {
		var icon string
		switch result.Status {
		case statusPass:
			icon = "✓"
			statusPassed++
		case statusWarning:
			icon = "⚠"
			statusWarnings++
		case statusFail:
			icon = "✗"
			statusFailed++
		}

		fmt.Printf("%s %s\n", icon, result.Check)
		fmt.Printf("  %s\n", result.Message)
		if result.Details != "" {
			fmt.Printf("  %s\n", result.Details)
		}
		fmt.Println()
	}

	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("Summary: %d statusPassed, %d statusWarnings, %d statusFailed\n", statusPassed, statusWarnings, statusFailed)

	if statusFailed > 0 {
		fmt.Println("\n⚠  Some checks statusFailed. Review the issues above.")
		return fmt.Errorf("health check statusFailed")
	}

	if statusWarnings > 0 {
		fmt.Println("\n⚠  Some checks have statusWarnings. Review recommended.")
	} else {
		fmt.Println("\n✓ All checks statusPassed!")
	}

	return nil
}

// outputDoctorJSON prints doctor results in JSON format.
func outputDoctorJSON(results []DoctorResult) error {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("statusFailed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}
