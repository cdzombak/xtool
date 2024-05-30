package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/google/subcommands"
)

var Version = "<dev>"

type versionCmd struct{}

func main() {
	ctx := context.Background()
	ctx = CtxWthErrPrintf(ctx, color.New(color.FgRed).PrintfFunc())
	ctx = CtxWthErrPrintln(ctx, color.New(color.FgRed).PrintlnFunc())

	cfg, err := buildAppConfig(ctx)
	if err != nil {
		ErrPrintf(ctx, "error getting app config: %s\n", err)
		os.Exit(int(subcommands.ExitFailure))
	}
	ctx = CtxWthAppConfig(ctx, cfg)

	subcommands.Register(subcommands.HelpCommand(), "")
	subcommands.Register(&versionCmd{}, "")
	subcommands.Register(&installCmd{}, "")
	subcommands.Register(&camswapCmd{}, "EXIF modification")
	subcommands.Register(&rmlocCmd{}, "EXIF modification")
	subcommands.Register(&inspectCmd{}, "EXIF inspection")
	subcommands.Register(&neatImgCmd{}, "noise reduction")
	subcommands.Register(&x3fJpgCmd{}, "Sigma X3F")

	flag.Parse()

	os.Exit(int(subcommands.Execute(ctx)))
}

func (*versionCmd) Name() string               { return "version" }
func (*versionCmd) Synopsis() string           { return "Print version and other information." }
func (p *versionCmd) SetFlags(_ *flag.FlagSet) {}

func (*versionCmd) Usage() string {
	return `version:
  Prints version, build, and other information about xtool.
`
}

func (p *versionCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	boldWhitePrintf := color.New(color.Bold, color.FgWhite).PrintfFunc()
	boldWhitePrintf("xtool %s\n", Version)
	fmt.Println(color.CyanString("https://www.github.com/cdzombak/xtool"))
	fmt.Printf(
		"a photo workflow tool by chris dzombak <%s>\n",
		color.CyanString("https://www.dzombak.com"),
	)
	fmt.Println()
	fmt.Printf(
		"embedded x3f_extract: %s (installable via `%s`)\n",
		X3fExtractVersion,
		color.MagentaString("xtool install -x3f-extract"),
	)
	fmt.Println(color.CyanString("https://github.com/Kalpanika/x3f"))
	fmt.Println()
	fmt.Printf(
		"run `%s` for usage.\n",
		color.MagentaString("xtool help"),
	)
	fmt.Println()

	return subcommands.ExitSuccess
}
