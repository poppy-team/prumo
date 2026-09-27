package main

import (
	"context"
	"fmt"

	"github.com/raillen/prumo/internal/autoupdate"
	"github.com/raillen/prumo/internal/protocol"
)

func runUpgrade(asJSON bool, explicitHome string, args []string) int {
	checkOnly := hasFlag(args, "--check")
	force := hasFlag(args, "--force")
	dryRun := hasFlag(args, "--dry-run")
	skipCache := hasFlag(args, "--no-cache")

	targetVersion, _, _ := flag(args, "--version")
	customRepo, _, _ := flag(args, "--repo")
	customToken, _, _ := flag(args, "--token")

	home, err := installationHome(explicitHome)
	if err != nil {
		return serviceError(asJSON, err)
	}

	repo := autoupdate.DefaultRepository
	if customRepo != "" {
		repo = customRepo
	}

	checkOpts := autoupdate.CheckOptions{
		Repository:     repo,
		CurrentVersion: protocol.CLIVersion,
		Token:          customToken,
		SkipCache:      skipCache,
		CacheDir:       home + "/cache",
	}

	ctx := context.Background()

	if checkOnly {
		res, err := autoupdate.Check(ctx, checkOpts)
		if err != nil {
			return serviceError(asJSON, err)
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(res))
		}

		fmt.Printf("Current version: %s\n", res.CurrentVersion)
		fmt.Printf("Latest version:  %s\n", res.LatestVersion)
		if res.HasUpdate {
			fmt.Printf("A new version of Prumo is available: v%s -> v%s\n", res.CurrentVersion, res.LatestVersion)
			fmt.Println("Run 'prumo upgrade' to install it.")
		} else {
			fmt.Println("Prumo is up to date.")
		}
		return exitOK
	}

	updateOpts := autoupdate.UpdateOptions{
		CheckOptions:  checkOpts,
		TargetVersion: targetVersion,
		Force:         force,
		DryRun:        dryRun,
	}

	if !asJSON {
		fmt.Printf("Checking for updates (current: v%s)...\n", protocol.CLIVersion)
	}

	res, err := autoupdate.Execute(ctx, updateOpts)
	if err != nil {
		return serviceError(asJSON, err)
	}

	if asJSON {
		return printEnvelope(protocol.OkEnvelope(res))
	}

	if res.PreviousVersion == res.UpdatedVersion && !force {
		fmt.Println(res.Message)
		return exitOK
	}

	fmt.Println(res.Message)
	if res.BackupPath != "" {
		fmt.Printf("Backup saved to: %s\n", res.BackupPath)
	}
	if res.VerifiedSHA256 != "" {
		fmt.Printf("Verified SHA256: %s\n", res.VerifiedSHA256)
	}

	// Trigger setup after upgrade to update catalogs and connectors
	if !dryRun {
		_ = runSetup(false, home)
	}

	return exitOK
}
