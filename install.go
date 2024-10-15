package main

import (
	"bytes"
	"context"
	_ "embed"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/codeclysm/extract/v4"
	"github.com/google/subcommands"
)

//go:embed x3f_extract
var x3fExtractBin []byte
var X3fExtractVersion = "<unknown>"

//go:embed applescript-embed.tar
var applescriptsArchive []byte

func LocalX3fExtractPath() string {
	return filepath.Join(MustUserHomeDir(), ".local", "bin.xtool", "x3f_extract")
}

type installCmd struct {
	x3fExtract   bool
	applescripts bool
}

func (*installCmd) Name() string { return "install" }
func (*installCmd) Synopsis() string {
	return "Install or update optional components/support tools for the program."
}

func (*installCmd) Usage() string {
	return `install [-scripts] [-x3f-extract]:
  Install or update optional components/support tools for the program.
`
}

func (p *installCmd) SetFlags(f *flag.FlagSet) {
	f.BoolVar(&p.x3fExtract, "x3f-extract", false, fmt.Sprintf("Install x3f_extract in ~/.local/bin.xtool. (version = %s)", X3fExtractVersion))
	f.BoolVar(&p.applescripts, "scripts", false, "Install AppleScript wrappers in ~/Library/Scripts.")
}

func (p *installCmd) Execute(ctx context.Context, _ *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if !p.x3fExtract && !p.applescripts {
		ErrPrintln(ctx, "no components selected for installation (must pass at least one of -x3f-extract or -scripts)")
		return subcommands.ExitUsageError
	}

	if p.x3fExtract {
		if err := installX3fExtract(ctx); err != nil {
			ErrPrint(ctx, err)
			return subcommands.ExitFailure
		}
	}

	if p.applescripts {
		if err := installApplescripts(ctx); err != nil {
			ErrPrint(ctx, err)
			return subcommands.ExitFailure
		}
	}

	return subcommands.ExitSuccess
}

func installX3fExtract(_ context.Context) error {
	fmt.Printf("Installing x3f_extract to %s ...", LocalX3fExtractPath())

	if err := os.MkdirAll(filepath.Dir(LocalX3fExtractPath()), 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(LocalX3fExtractPath(), x3fExtractBin, 0755); err != nil {
		return fmt.Errorf("failed to write x3f_extract file: %w", err)
	}

	fmt.Println("done.")
	return nil
}

func installApplescripts(ctx context.Context) error {
	workDir, err := os.MkdirTemp("", fmt.Sprintf("xtool-applescripts-%s-*", Version))
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	//goland:noinspection GoUnhandledErrorResult
	defer os.RemoveAll(workDir)

	buffer := bytes.NewBuffer(applescriptsArchive)
	err = extract.Tar(ctx, buffer, workDir, nil)
	if err != nil {
		return fmt.Errorf("failed to extract AppleScripts archive to '%s': %w", workDir, err)
	}

	chmodCmd := exec.CommandContext(ctx,
		"chmod", "u+x",
		filepath.Join(workDir, "./install.sh"),
		filepath.Join(workDir, "./restore-resources.sh"),
	)
	chmodCmd.Dir = workDir
	out, err := chmodCmd.CombinedOutput()
	if err != nil {
		ErrPrintf(ctx, "failed to make install scripts executable:\n%s\n", out)
		return err
	}

	installCmd := exec.CommandContext(ctx, "./install.sh")
	installCmd.Dir = workDir
	installCmd.Env = append(os.Environ(), "XTOOL_VERSION="+Version)
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	err = installCmd.Run()
	if err != nil {
		return err
	}

	fmt.Println("done.")

	return nil
}
