package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/fatih/color"
	"github.com/google/subcommands"
)

type x3fJpgCmd struct {
	outDir    string
	verbose   bool
	verbose2  bool
	appConfig AppConfig
}

func (*x3fJpgCmd) Name() string { return "x3fjpg" }
func (*x3fJpgCmd) Synopsis() string {
	return "Extract embedded JPEG from Sigma X3F files."
}

func (*x3fJpgCmd) Usage() string {
	return `x3fjpg [-d out_dir] [-v|-vv] file1.x3f [file2.x3f ...]:
  Extract the embedded JPEG from the given Sigma X3F RAW image files.
`
}

func (p *x3fJpgCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&p.outDir, "d", "", "Write extracted JPEGs to this directory.")
	f.BoolVar(&p.verbose, "v", false, "Print full x3f_extract output for each image.")
	f.BoolVar(&p.verbose2, "vv", false, "Print x3f_extract commands and their full output.")
}

func (p *x3fJpgCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if p.verbose2 {
		p.verbose = true
	}

	if len(f.Args()) == 0 {
		f.Usage()
		return subcommands.ExitFailure
	}

	p.appConfig = GetAppConfig()

	// Fallback to finding x3f_extract in the path if it wasn't specified in a config:
	if p.appConfig.X3fBin == "" {
		x3fBin, err := exec.LookPath("x3f_extract")
		if err != nil {
			fmt.Println("x3f_bin was not specified in config and x3f_extract is missing from $PATH")
			fmt.Printf("$PATH search failed with: %s\n", err)
			os.Exit(1)
		}
		p.appConfig.X3fBin = x3fBin
	}

	x3fArgs := []string{"-jpg"}

	if p.outDir != "" {
		x3fArgs = append(x3fArgs, "-o", p.outDir)

		// prep output directory:
		err := os.MkdirAll(p.outDir, 0777)
		if err != nil {
			fmt.Printf("failed to ensure '%s' exists: %s\n", p.outDir, err)
			return subcommands.ExitFailure
		}
	}
	if !p.verbose {
		x3fArgs = append(x3fArgs, "-q")
	} else if p.verbose2 {
		x3fArgs = append(x3fArgs, "-v")
	}

	successes, failures := X3fJpgProcess(
		x3fArgs,
		f.Args(),
		p.appConfig,
		p.verbose,
		p.verbose2,
	)

	boldWhitePrintf := color.New(color.Bold, color.FgWhite).PrintfFunc()
	boldRedPrintf := color.New(color.Bold, color.FgRed).PrintfFunc()

	boldWhitePrintf("\nx3fjpg: successfully extracted %d images.\n", len(successes))

	if len(failures) != 0 {
		boldRedPrintf("Errors:\n")
		for filename, err := range failures {
			fmt.Printf("- %s %s\n", color.MagentaString("%s:", filename), err)
		}
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}

// X3fJpgProcess returns list of files successfully processed, and map of filename -> error.
func X3fJpgProcess(args []string, files []string, appConfig AppConfig, verbose, verbose2 bool) ([]string, map[string]error) {
	errorPrintln := color.New(color.FgRed).PrintlnFunc()

	var successes []string
	errs := make(map[string]error)

	for _, imgFilename := range files {
		fmt.Printf("%s ...\n", imgFilename)

		fullArgs := make([]string, len(args)+1)
		copy(fullArgs, args)
		fullArgs[len(args)] = imgFilename

		if verbose2 {
			fmt.Printf("%s %s\n", appConfig.X3fBin, strings.Join(fullArgs, " "))
		}

		cmdOut, err := RunCmd(appConfig.X3fBin, fullArgs)
		if err != nil {
			errs[imgFilename] = err
			errorPrintln(errs[imgFilename].Error())
			continue
		}
		if verbose {
			fmt.Println(cmdOut)
		}

		successes = append(successes, imgFilename)
	}

	return successes, errs
}
