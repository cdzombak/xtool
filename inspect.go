package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"sort"

	"github.com/fatih/color"
	"github.com/google/subcommands"
)

type inspectCmd struct {
	location  bool
	swap      bool
	appConfig AppConfig
}

func (*inspectCmd) Name() string     { return "inspect" }
func (*inspectCmd) Synopsis() string { return "Inspect image files for GPS or camera-swap data." }

func (*inspectCmd) Usage() string {
	return `inspect -l|-s file1.jpg [file2.nef ...]:
  Inspects the given image files for GPS or camera-swap data.
`
}

func (p *inspectCmd) SetFlags(f *flag.FlagSet) {
	f.BoolVar(&p.location, "l", false, "Inspect image files for location/GPS data.")
	f.BoolVar(&p.location, "g", false, "Inspect image files for location/GPS data (alias for -l).")
	f.BoolVar(&p.swap, "s", false, "Inspect image files for camera-swap data.")
}

func (p *inspectCmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if len(f.Args()) == 0 {
		f.Usage()
		return subcommands.ExitUsageError
	}

	if !p.swap && !p.location {
		p.swap = true
		p.location = true
	}

	p.appConfig = AppConfigFromCtx(ctx)

	exiftoolConfigFilename, err := getExiftoolConfigFileName()
	if err != nil {
		ErrPrint(ctx, err)
		return subcommands.ExitFailure
	}
	//goland:noinspection GoUnhandledErrorResult
	defer func() { _ = os.Remove(exiftoolConfigFilename) }()

	swapExiftoolArgs := []string{"-j", "-f", "-Model", "-XtoolOriginalCameraModel"}
	locationExiftoolArgs := []string{"-j", "-gps*"}

	fmt.Println()

	boldWhitePrintf := color.New(color.Bold, color.FgWhite).PrintfFunc()
	boldGreenPrintf := color.New(color.FgGreen).PrintfFunc()

	gpsTagAllowlist := []string{"GPSVersionID", "SourceFile"}

	for _, imgFilename := range f.Args() {
		boldWhitePrintf("%s ...\n", imgFilename)

		if p.swap {
			fullArgs := make([]string, len(swapExiftoolArgs)+1)
			copy(fullArgs, swapExiftoolArgs)
			fullArgs[len(swapExiftoolArgs)] = imgFilename

			cmd := exec.Command(p.appConfig.ExiftoolBin, fullArgs...)
			cmdOut, err := cmd.CombinedOutput()
			if err != nil {
				fmt.Printf("\tfailed to run exiftool: %s\n\n", err)
				continue
			}
			var result []map[string]string
			err = json.Unmarshal(cmdOut, &result)
			if err != nil {
				fmt.Printf("\tfailed to parse exiftool result as JSON: %s\n\n", err)
				continue
			}
			if len(result) != 1 {
				fmt.Printf("\tinvalid exiftool output: expected 1 item, got %d\n\n", len(result))
				continue
			}

			metadata := result[0]
			if swapped, ok := metadata["XtoolOriginalCameraModel"]; ok && swapped != "-" {
				fmt.Printf("\t%s %s\n", color.MagentaString("Original Camera Model:"), swapped)
				if model, ok := metadata["Model"]; ok {
					fmt.Printf("\t%s %s\n", color.MagentaString("Swapped Camera Model:"), model)
				}
			} else {
				boldGreenPrintf("\t✔ No camera swap metadata.\n")
				if model, ok := metadata["Model"]; ok {
					fmt.Printf("\t%s %s\n", color.MagentaString("Camera Model:"), model)
				}
			}
			fmt.Println()
		}

		if p.location {
			fullArgs := make([]string, len(locationExiftoolArgs)+1)
			copy(fullArgs, locationExiftoolArgs)
			fullArgs[len(locationExiftoolArgs)] = imgFilename

			cmd := exec.Command(p.appConfig.ExiftoolBin, fullArgs...)
			cmdOut, err := cmd.CombinedOutput()
			if err != nil {
				fmt.Printf("\tfailed to run exiftool: %s\n\n", err)
				continue
			}
			var result []map[string]interface{}
			err = json.Unmarshal(cmdOut, &result)
			if err != nil {
				fmt.Printf("\tfailed to parse exiftool result as JSON: %s\n\n", err)
				continue
			}
			if len(result) != 1 {
				fmt.Printf("\tinvalid exiftool output: expected 1 item, got %d\n\n", len(result))
				continue
			}

			metadata := result[0]
			for _, k := range gpsTagAllowlist {
				delete(metadata, k)
			}
			if len(metadata) == 0 {
				boldGreenPrintf("\t✔ No GPS metadata.\n")
			} else {
				keys := make([]string, 0, len(metadata))
				for k := range metadata {
					keys = append(keys, k)
				}
				sort.Strings(keys)

				for _, k := range keys {
					fmt.Printf("\t%s %s\n", color.MagentaString("%s:", k), metadata[k])
				}
			}
			fmt.Println()
		}
	}

	return subcommands.ExitSuccess
}
