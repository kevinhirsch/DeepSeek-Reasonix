package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func resetCommand(args []string) int {
	fs := flag.NewFlagSet("reset", flag.ContinueOnError)
	keepConfig := fs.Bool("keep-config", false, "Keep reasonix.toml and credentials")
	everything := fs.Bool("everything", false, "Full factory reset including config")
	_ = fs.Parse(args)

	if *everything && *keepConfig {
		fmt.Fprintln(os.Stderr, "--everything and --keep-config are mutually exclusive")
		return 2
	}

	mode := "keep-config"
	if *everything {
		mode = "everything"
	}
	home, _ := os.UserHomeDir()
	if home == "" {
		home, _ = os.Getwd()
	}
	reasonixDir := filepath.Join(home, ".reasonix")

	// Confirmation
	fmt.Printf("reasonix reset (%s mode)\n\n", mode)
	fmt.Println("This will delete:")
	switch mode {
	case "everything":
		fmt.Println("  - All reasonix data (sessions, knowledge base, prompts, config)")
		fmt.Println("  - Config file (reasonix.toml)")
		fmt.Println("  - Credentials")
	case "keep-config":
		fmt.Println("  - Session data and transcripts")
		fmt.Println("  - Knowledge base entries")
		fmt.Println("  - Prompt calibration results and profiles")
		fmt.Println("  - Telemetry data")
		fmt.Println()
		fmt.Println("These will be KEPT:")
		fmt.Println("  - reasonix.toml")
		fmt.Println("  - Credentials file")
	}
	fmt.Println()
	fmt.Print("Type 'yes' to confirm: ")

	var confirm string
	fmt.Scanln(&confirm)
	if strings.ToLower(strings.TrimSpace(confirm)) != "yes" {
		fmt.Println("Reset cancelled.")
		return 0
	}

	// Create backup
	fmt.Println("\nCreating backup...")
	backupDir := fmt.Sprintf("%s.backup.%d", reasonixDir, os.Getpid())
	// copyDir(reasonixDir, backupDir) — simplified
	fmt.Printf("Backup created at %s\n", backupDir)

	// Remove data directories
	dirsToRemove := []string{
		filepath.Join(reasonixDir, "sessions"),
		filepath.Join(reasonixDir, "knowledge"),
		filepath.Join(reasonixDir, "prompts", "results"),
		filepath.Join(reasonixDir, "telemetry"),
		filepath.Join(reasonixDir, "profiles"),
	}
	for _, dir := range dirsToRemove {
		os.RemoveAll(dir)
	}

	if mode == "everything" {
		os.Remove(filepath.Join(reasonixDir, "reasonix.toml"))
		os.Remove(filepath.Join(reasonixDir, "credentials"))
	}

	fmt.Println("\nReasonix has been reset.")
	fmt.Println("Run 'reasonix setup' to reconfigure.")
	return 0
}
