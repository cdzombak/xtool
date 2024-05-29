package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/google/subcommands"
)

type camswapCmd struct {
	restore     bool
	suffix      bool
	outDir      string
	verbose     bool
	verbose2    bool
	newCamModel string
	appConfig   AppConfig
}

func (*camswapCmd) Name() string     { return "camswap" }
func (*camswapCmd) Synopsis() string { return "Swap in a different camera name." }

func (*camswapCmd) Usage() string {
	return `camswap [-c CAM_MODEL|-c CAM_ALIAS] [-r] [-s] [-d out_dir] [-v|-vv] file1.jpg [file2.nef ...]:
  Swaps a different camera model into the given photos' EXIF data.
  Persists the original name in an XMP attribute for restoration with the -r flag.
  Exactly one of (-c, -r) is required.
`
}

func (p *camswapCmd) SetFlags(f *flag.FlagSet) {
	f.BoolVar(&p.suffix, "s", false, "Write modified images to new files named with a suffix derived from the camera name/alias, rather than to the originals.")
	f.StringVar(&p.outDir, "d", "", "Write modified images to this directory.")
	f.BoolVar(&p.verbose, "v", false, "Print full exiftool output for each image.")
	f.BoolVar(&p.verbose2, "vv", false, "Print exiftool commands and full exiftool output.")

	f.StringVar(&p.newCamModel, "c", "", "Camera model to swap in (or alias defined in camswap_aliases).")
	f.BoolVar(&p.restore, "r", false, "Restore the original camera name from xtool's XMP attribute.")
}

func (p *camswapCmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if p.verbose2 {
		p.verbose = true
	}

	if len(f.Args()) == 0 || (!p.restore && p.newCamModel == "") || (p.restore && p.newCamModel != "") {
		f.Usage()
		return subcommands.ExitUsageError
	}

	p.appConfig = AppConfigFromCtx(ctx)

	exiftoolConfigFilename, err := getExiftoolConfigFileName()
	if err != nil {
		ErrPrint(ctx, err)
		return subcommands.ExitFailure
	}
	//goland:noinspection GoUnhandledErrorResult
	defer os.Remove(exiftoolConfigFilename)

	var exiftoolArgs []string
	if p.restore {
		exiftoolArgs = []string{
			"-config", exiftoolConfigFilename,
			"-Model<XtoolOriginalCameraModel",
			"-XtoolOriginalCameraModel=",
			"-if", "$XtoolOriginalCameraModel",
		}

		if p.outDir != "" && p.suffix {
			exiftoolArgs = append(exiftoolArgs, "-o", fmt.Sprintf("%s%s%%f_unswap.%%e", p.outDir, string(os.PathSeparator)))
		} else if p.suffix {
			exiftoolArgs = append(exiftoolArgs, "-o", "%d%f_unswap.%e")
		} else if p.outDir != "" {
			exiftoolArgs = append(exiftoolArgs, "-o", fmt.Sprintf("%s%s", p.outDir, string(os.PathSeparator)))
		}
	} else {
		newModel := p.newCamModel
		if p.appConfig.CamswapAliases[p.newCamModel] != "" {
			newModel = p.appConfig.CamswapAliases[p.newCamModel]
		}

		exiftoolArgs = []string{
			"-config", exiftoolConfigFilename,
			"-XtoolOriginalCameraModel<Model",
			fmt.Sprintf("-Model=%s", newModel),
			"-if", "not $XtoolOriginalCameraModel",
		}

		suffixSafeCamModel := strings.Replace(p.newCamModel, " ", "-", -1)
		if p.outDir != "" && p.suffix {
			exiftoolArgs = append(exiftoolArgs, "-o", fmt.Sprintf("%s%s%%d%%f_%s.%%e", p.outDir, string(os.PathSeparator), suffixSafeCamModel))
		} else if p.suffix {
			exiftoolArgs = append(exiftoolArgs, "-o", fmt.Sprintf("%%d%%f_%s.%%e", suffixSafeCamModel))
		} else if p.outDir != "" {
			exiftoolArgs = append(exiftoolArgs, "-o", fmt.Sprintf("%s%s", p.outDir, string(os.PathSeparator)))
		}
	}

	successes, failures := ExiftoolProcess(
		ctx,
		exiftoolArgs,
		f.Args(),
		p.appConfig,
		p.verbose,
		p.verbose2,
	)

	boldWhitePrintf := color.New(color.Bold, color.FgWhite).PrintfFunc()
	boldRedPrintf := color.New(color.Bold, color.FgRed).PrintfFunc()

	boldWhitePrintf("\ncamswap: successfully processed %d images.\n", len(successes))

	if len(failures) != 0 {
		boldRedPrintf("Errors:\n")
		for filename, err := range failures {
			errString := err.Error()
			if strings.Contains(err.Error(), "failed condition") {
				if p.restore {
					errString = "no camera swap metadata attached"
				} else {
					errString = "has already been camswapped"
				}
			}
			fmt.Printf("- %s %s\n", color.MagentaString("%s:", filename), errString)
		}
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}

func getExiftoolConfigFileName() (string, error) {
	exiftoolConfigFile, err := os.CreateTemp("", "xtool_xmp")
	if err != nil {
		return "", fmt.Errorf("failed to create exiftool XMP config file: %w", err)
	}
	exiftoolConfigFilename := exiftoolConfigFile.Name()
	if _, err = exiftoolConfigFile.Write([]byte(exiftoolXtoolXmpConfig)); err != nil {
		return "", fmt.Errorf("failed to write exiftool XMP config file: %w", err)
	}
	if err = exiftoolConfigFile.Close(); err != nil {
		return "", fmt.Errorf("failed to close exiftool XMP config file: %w", err)
	}
	return exiftoolConfigFilename, nil
}

const exiftoolXtoolXmpConfig = `
%Image::ExifTool::UserDefined = (
    'Image::ExifTool::XMP::xmp' => {
        XtoolOriginalCameraModel => { },
    },
);

1;
`
