package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/google/subcommands"
)

// TODO(cdzombak): Error flow throughout the application could really use cleanup
//                 This code is littered with os.Exit(1) when ideally errors would bubble up to the top level.

var version = "<dev build>"

type versionCmd struct{}

func main() {

	subcommands.Register(subcommands.HelpCommand(), "")
	subcommands.Register(&versionCmd{}, "")
	subcommands.Register(&camswapCmd{}, "EXIF modification")
	subcommands.Register(&rmlocCmd{}, "EXIF modification")
	subcommands.Register(&inspectCmd{}, "EXIF inspection")

	// SOMEDAY(cdzombak): x3f extract, etc.
	// https://github.com/Kalpanika/x3f
	// (but that doesn't build natively on macOS, so ... not dealing with it now)

	flag.Parse()
	ctx := context.Background()
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

	boldWhitePrintf("xtool %s\n", version)
	fmt.Println(color.CyanString("https://www.github.com/cdzombak/xtool"))
	fmt.Println()
	fmt.Println("a photo workflow tool by chris dzombak https://www.dzombak.com")
	fmt.Println()
	fmt.Println("(particularly useful for tricking DxO into")
	fmt.Println(" processing files from unsupported cameras)")
	fmt.Println()
	fmt.Println(color.MagentaString("run `xtool help` for usage."))
	fmt.Println()

	return subcommands.ExitSuccess
}
