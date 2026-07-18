package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/EdgarOrtegaRamirez/composeforge/pkg/analyzer"
	"github.com/EdgarOrtegaRamirez/composeforge/pkg/differ"
	"github.com/EdgarOrtegaRamirez/composeforge/pkg/merger"
	"github.com/EdgarOrtegaRamirez/composeforge/pkg/parser"
	"github.com/EdgarOrtegaRamirez/composeforge/pkg/validator"
	"github.com/spf13/cobra"
)

var version = "0.1.0"

func main() {
	rootCmd := &cobra.Command{
		Use:   "composeforge",
		Short: "Docker Compose analysis & management toolkit",
		Long:  "A comprehensive CLI tool for analyzing, validating, diffing, and managing Docker Compose files.",
	}

	rootCmd.AddCommand(
		newValidateCmd(),
		newAnalyzeCmd(),
		newDiffCmd(),
		newMergeCmd(),
		newVersionCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func newValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate [file...]",
		Short: "Validate docker-compose files",
		Long:  "Validate docker-compose files for errors, security issues, and best practices.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runValidate(args)
		},
	}
	return cmd
}

func runValidate(files []string) error {
	totalErrors := 0
	totalWarnings := 0

	for _, file := range files {
		cf, err := parser.ParseFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing %s: %v\n", file, err)
			totalErrors++
			continue
		}

		result := validator.Validate(cf)

		if len(result.Issues) == 0 {
			fmt.Printf("✓ %s: valid\n", file)
			continue
		}

		fmt.Printf("\n%s:\n", file)
		for _, issue := range result.Issues {
			prefix := "  "
			switch issue.Severity {
			case validator.SeverityCritical:
				prefix = "  🔴 "
			case validator.SeverityError:
				prefix = "  ❌ "
			case validator.SeverityWarning:
				prefix = "  ⚠️  "
			case validator.SeverityInfo:
				prefix = "  ℹ️  "
			}
			svc := ""
			if issue.Service != "" {
				svc = fmt.Sprintf("[%s] ", issue.Service)
			}
			fmt.Printf("%s%s%s: %s\n", prefix, svc, issue.Category, issue.Message)
		}

		fmt.Printf("\n  Summary: %d errors, %d warnings, %d info\n",
			result.ErrorCount, result.WarningCount, result.InfoCount)

		totalErrors += result.ErrorCount
		totalWarnings += result.WarningCount
	}

	fmt.Printf("\n═══\n")
	fmt.Printf("Total: %d errors, %d warnings across %d file(s)\n", totalErrors, totalWarnings, len(files))

	if totalErrors > 0 {
		return fmt.Errorf("validation failed with %d error(s)", totalErrors)
	}
	return nil
}

func newAnalyzeCmd() *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "analyze [file]",
		Short: "Analyze a docker-compose file",
		Long:  "Perform comprehensive analysis of a docker-compose file including dependency graph, resource usage, and network topology.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAnalyze(args[0], output)
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format: text, json")

	return cmd
}

func runAnalyze(file, output string) error {
	cf, err := parser.ParseFile(file)
	if err != nil {
		return fmt.Errorf("error parsing file: %w", err)
	}

	analysis := analyzer.Analyze(cf)

	switch output {
	case "json":
		// JSON output
		fmt.Printf("%s\n", formatAnalysisJSON(analysis))
	default:
		// Text output
		fmt.Println("═══ ComposeForge Analysis ═══")
		fmt.Println()
		fmt.Print(analysis.Summary())
		fmt.Println()

		fmt.Println("── Build Order ──")
		for i, svc := range analysis.BuildOrder {
			fmt.Printf("  %d. %s\n", i+1, svc)
		}
		fmt.Println()

		fmt.Println("── Dependency Tree ──")
		fmt.Print(analyzer.FormatDependencyTree(analysis))
		fmt.Println()

		fmt.Println("── Network Graph ──")
		fmt.Print(analyzer.FormatNetworkGraph(analysis))

		if len(analysis.Warnings) > 0 {
			fmt.Println()
			fmt.Println("── Warnings ──")
			for _, w := range analysis.Warnings {
				fmt.Printf("  ⚠️  %s\n", w)
			}
		}
	}

	return nil
}

func formatAnalysisJSON(a *analyzer.ComposeAnalysis) string {
	// Simple JSON formatting
	var sb strings.Builder
	sb.WriteString("{\n")
	sb.WriteString(fmt.Sprintf("  \"total_services\": %d,\n", len(a.Services)))
	sb.WriteString(fmt.Sprintf("  \"total_networks\": %d,\n", a.TotalNetworks))
	sb.WriteString(fmt.Sprintf("  \"total_volumes\": %d,\n", a.TotalVolumes))
	sb.WriteString(fmt.Sprintf("  \"total_secrets\": %d,\n", a.TotalSecrets))
	sb.WriteString(fmt.Sprintf("  \"total_ports\": %d,\n", a.TotalPorts))
	sb.WriteString(fmt.Sprintf("  \"warnings\": %d,\n", len(a.Warnings)))
	sb.WriteString("  \"services\": [\n")
	for i, svc := range a.Services {
		if i > 0 {
			sb.WriteString(",\n")
		}
		sb.WriteString(fmt.Sprintf("    {\"name\": \"%s\", \"image\": \"%s\", \"depth\": %d}",
			svc.Name, svc.Image, svc.Depth))
	}
	sb.WriteString("\n  ],\n")
	sb.WriteString("  \"build_order\": [")
	for i, svc := range a.BuildOrder {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("\"%s\"", svc))
	}
	sb.WriteString("]\n}")
	return sb.String()
}

func newDiffCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff [base] [override]",
		Short: "Diff two docker-compose files",
		Long:  "Compare two docker-compose files and show the differences.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDiff(args[0], args[1])
		},
	}
	return cmd
}

func runDiff(baseFile, overrideFile string) error {
	base, err := parser.ParseFile(baseFile)
	if err != nil {
		return fmt.Errorf("error parsing base file: %w", err)
	}

	override, err := parser.ParseFile(overrideFile)
	if err != nil {
		return fmt.Errorf("error parsing override file: %w", err)
	}

	diff := differ.Diff(base, override)
	output := differ.FormatDiff(diff)
	fmt.Println(output)

	if !diff.HasChanges {
		return nil
	}

	fmt.Printf("\n═══\n")
	fmt.Printf("Changes: %d added, %d removed, %d modified\n",
		len(diff.AddedServices), len(diff.RemovedServices), len(diff.ModifiedServices))

	return nil
}

func newMergeCmd() *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "merge [base] [override...]",
		Short: "Merge docker-compose files",
		Long:  "Merge multiple docker-compose files, with later files overriding earlier ones.",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMerge(args, output)
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file (default: stdout)")

	return cmd
}

func runMerge(files []string, outputFile string) error {
	var composeFiles []*parser.ComposeFile

	for _, file := range files {
		cf, err := parser.ParseFile(file)
		if err != nil {
			return fmt.Errorf("error parsing %s: %w", file, err)
		}
		composeFiles = append(composeFiles, cf)
	}

	merged, err := merger.Merge(composeFiles...)
	if err != nil {
		return fmt.Errorf("error merging: %w", err)
	}

	// Simple YAML output
	output := formatComposeYAML(merged)

	if outputFile != "" {
		if err := os.WriteFile(outputFile, []byte(output), 0644); err != nil {
			return fmt.Errorf("error writing output: %w", err)
		}
		fmt.Printf("Merged output written to %s\n", outputFile)
	} else {
		fmt.Print(output)
	}

	return nil
}

func formatComposeYAML(cf *parser.ComposeFile) string {
	var sb strings.Builder

	if cf.Version != "" {
		sb.WriteString(fmt.Sprintf("version: '%s'\n", cf.Version))
	}
	if cf.Name != "" {
		sb.WriteString(fmt.Sprintf("name: %s\n", cf.Name))
	}

	sb.WriteString("\nservices:\n")
	for name, svc := range cf.Services {
		sb.WriteString(fmt.Sprintf("  %s:\n", name))
		if svc.Image != "" {
			sb.WriteString(fmt.Sprintf("    image: %s\n", svc.Image))
		}
		if len(svc.Ports) > 0 {
			sb.WriteString("    ports:\n")
			for _, port := range svc.Ports {
				sb.WriteString(fmt.Sprintf("      - \"%s\"\n", port))
			}
		}
		if len(svc.Volumes) > 0 {
			sb.WriteString("    volumes:\n")
			for _, vol := range svc.Volumes {
				sb.WriteString(fmt.Sprintf("      - %s\n", vol))
			}
		}
		env := parser.NormalizeEnvironment(svc.Environment)
		if len(env) > 0 {
			sb.WriteString("    environment:\n")
			for k, v := range env {
				sb.WriteString(fmt.Sprintf("      %s: %s\n", k, v))
			}
		}
		if svc.Restart != "" {
			sb.WriteString(fmt.Sprintf("    restart: %s\n", svc.Restart))
		}
	}

	return sb.String()
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("composeforge v%s\n", version)
		},
	}
}
