package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/google/subcommands"
)

type rmlocCmd struct {
	suffix    bool
	outDir    string
	verbose   bool
	verbose2  bool
	appConfig AppConfig
}

func (*rmlocCmd) Name() string     { return "rmloc" }
func (*rmlocCmd) Synopsis() string { return "Remove all GPS metadata." }

func (*rmlocCmd) Usage() string {
	return `rmloc [-s] [-v|-vv] file1.jpg [file2.nef ...]:
  Removes all GPS data from the given files.
`
}

func (p *rmlocCmd) SetFlags(f *flag.FlagSet) {
	f.BoolVar(&p.suffix, "s", false, "Write modified images to new files named with the suffix _noGPS, rather than to the originals.")
	f.StringVar(&p.outDir, "d", "", "Write modified images to a subdirectory with this name.")
	f.BoolVar(&p.verbose, "v", false, "Print full exiftool output for each image.")
	f.BoolVar(&p.verbose2, "vv", false, "Print exiftool commands and full exiftool output.")
}

func (p *rmlocCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if p.verbose2 {
		p.verbose = true
	}

	if len(f.Args()) == 0 {
		f.Usage()
		return subcommands.ExitFailure
	}

	p.appConfig = GetAppConfig()

	exiftoolArgs := []string{"-gps*="}
	if p.outDir != "" && p.suffix {
		exiftoolArgs = append(exiftoolArgs, "-o", fmt.Sprintf("%s%s%%d%%f_noGPS.%%e", p.outDir, string(os.PathSeparator)))
	} else if p.suffix {
		exiftoolArgs = append(exiftoolArgs, "-o", "%d%f_noGPS.%e")
	} else if p.outDir != "" {
		exiftoolArgs = append(exiftoolArgs, "-o", fmt.Sprintf("%s%s", p.outDir, string(os.PathSeparator)))
	}

	successes, failures := ExiftoolProcess(
		exiftoolArgs,
		f.Args(),
		p.appConfig,
		p.verbose,
		p.verbose2,
	)

	boldWhitePrintf := color.New(color.Bold, color.FgWhite).PrintfFunc()
	boldRedPrintf := color.New(color.Bold, color.FgRed).PrintfFunc()

	boldWhitePrintf("\nrmloc: successfully processed %d images.\n", len(successes))

	if len(failures) != 0 {
		boldRedPrintf("Errors:\n")
		for filename, err := range failures {
			fmt.Printf("- %s %s\n", color.MagentaString("%s:", filename), err)
		}
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}
