package main

import (
	"context"
	"flag"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/google/subcommands"
)

const defaultNeatImageCLName = "NeatImage9CL"

type neatImgCmd struct {
	outDir     string
	jpgQuality int
	verbose    bool
	verbose2   bool
	appConfig  AppConfig
}

func (*neatImgCmd) Name() string { return "neatimg" }
func (*neatImgCmd) Synopsis() string {
	return "Denoise images with NeatImage."
}

func (*neatImgCmd) Usage() string {
	return `neatimg [-q jpg_quality] [-d out_dir] [-v|-vv] file1.jpg [file2.nef ...]:
  Denoise images with the NeatImage CLI tool. Uses Smart Profile. All other settings (eg. filename suffix, default preset) are controlled by the defaults in the Neat Image GUI settings.
`
}

func (p *neatImgCmd) SetFlags(f *flag.FlagSet) {
	f.IntVar(&p.jpgQuality, "q", 0, "Quality for JPEG compression. If not set here or in neat_image.default_jpg_quality, defaults to 80.")

	// note: this cmd refuses to overwrite files, so -s is implied and the defautl set in the Neat Image GUI settings is used.
	// it's not possible to write into a subdir with no suffix. I don't want to complicate this CLI with something I won't use.
	f.StringVar(&p.outDir, "d", "", "Write denoised images to this directory.")
	f.BoolVar(&p.verbose, "v", false, "Print full NeatImageCL output for each image.")
	f.BoolVar(&p.verbose2, "vv", false, "Print NeatImageCL commands and their full output.")
}

func (p *neatImgCmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if p.verbose2 {
		p.verbose = true
	}

	if len(f.Args()) == 0 {
		f.Usage()
		return subcommands.ExitUsageError
	}

	if p.jpgQuality < 0 || p.jpgQuality > 100 {
		fmt.Printf("invalid -q: '%d'\n", p.jpgQuality)
		return subcommands.ExitUsageError
	}

	p.appConfig = AppConfigFromCtx(ctx)

	// Fallback to finding neat image in the path if it wasn't specified in a config:
	if p.appConfig.NeatImage.NeatImageBin == "" {
		neatImagePath, err := exec.LookPath(defaultNeatImageCLName)
		if err != nil {
			ErrPrintln(ctx, "neat_image.neat_image_bin was not specified in config and is missing from $PATH")
			ErrPrintf(ctx, "$PATH search failed with: %s", err)
			return subcommands.ExitFailure
		}
		p.appConfig.NeatImage.NeatImageBin = neatImagePath
	}

	if p.appConfig.NeatImage.DefaultJpgQuality < 0 || p.appConfig.NeatImage.DefaultJpgQuality > 100 {
		ErrPrintf(ctx, "invalid neat_image.default_jpg_quality '%d'\n", p.appConfig.NeatImage.DefaultJpgQuality)
	}
	targetJpgQuality := p.jpgQuality
	if targetJpgQuality == 0 {
		targetJpgQuality = p.appConfig.NeatImage.DefaultJpgQuality
	}
	if targetJpgQuality == 0 {
		targetJpgQuality = 80
	}

	// uses smart profile, auto fine tune, and your default preset.
	neatImgArgs := []string{"--smart-profile", "--no-overwrite", "--preserve-meta", "--output-bitdepth=M"}
	if p.appConfig.NeatImage.ProfilesFolder != "" {
		pf, err := filepath.Abs(p.appConfig.NeatImage.ProfilesFolder)
		if err != nil {
			ErrPrintf(ctx, "could not get path to profiles folder '%s': %s\n", p.appConfig.NeatImage.ProfilesFolder, err)
			return subcommands.ExitFailure
		}
		neatImgArgs = append(neatImgArgs, fmt.Sprintf("--profile-folder=%s", pf))
	}

	if p.outDir != "" {
		neatImgArgs = append(neatImgArgs, fmt.Sprintf("--output-folder=%s", p.outDir))
	} else {
		neatImgArgs = append(neatImgArgs, "--output-to-input-folder")
	}

	successes, failures := NeatImageProcess(
		neatImgArgs,
		f.Args(),
		p.appConfig,
		p.verbose,
		p.verbose2,
		targetJpgQuality,
	)

	boldWhitePrintf := color.New(color.Bold, color.FgWhite).PrintfFunc()
	boldRedPrintf := color.New(color.Bold, color.FgRed).PrintfFunc()

	boldWhitePrintf("\nneatimg: successfully processed %d images.\n", len(successes))

	if len(failures) != 0 {
		boldRedPrintf("Errors:\n")
		for filename, err := range failures {
			fmt.Printf("- %s %s\n", color.MagentaString("%s:", filename), err)
		}
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}

// NeatImageProcess returns list of files successfully processed, and map of filename -> error.
func NeatImageProcess(args []string, files []string, appConfig AppConfig, verbose, verbose2 bool, jpgQuality int) ([]string, map[string]error) {
	errorPrintln := color.New(color.FgRed).PrintlnFunc()

	var successes []string
	errs := make(map[string]error)

	for _, imgFilename := range files {
		fmt.Printf("%s ...\n", imgFilename)

		// note: NeatImageCL <InputImage...> [<Profile>] [<Preset>] [<Output>] [<Log>]
		fullArgs := make([]string, len(args)+1)
		fullArgs[0] = imgFilename
		fullArgs = append(fullArgs, args...)

		inputExt := strings.ToLower(filepath.Ext(imgFilename))
		switch inputExt {
		case ".jpg", ".jpeg":
			fullArgs = append(fullArgs, "--output-format=JPG")
		case ".tif", ".tiff":
			fullArgs = append(fullArgs, "--output-format=TIF")
		case ".png":
			fullArgs = append(fullArgs, "--output-format=PNG")
		}

		if verbose2 {
			fmt.Printf("%s %s\n", appConfig.NeatImage.NeatImageBin, strings.Join(fullArgs, " "))
		}

		cmdOut, err := RunCmd(appConfig.NeatImage.NeatImageBin, fullArgs)
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
