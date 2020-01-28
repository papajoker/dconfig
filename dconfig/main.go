package main

import (
	"flag"

	//"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
)

var GitBranch string
var Version string
var BuildDate string
var GitID string

var OptionQuiet *bool
var OptionD *bool

func main() {

	os.Setenv("LANG", "C")
	OptionVersion := flag.Bool("v", false, "show Version")
	//OptionFiles := flag.Bool("f", false, "Get Modified Files")
	OptionD := flag.Bool("d", false, "Find Override Configuration (/*.d/*)")
	OptionQuiet := flag.Bool("q", false, "Quiet mode (with -f)")
	flag.Parse()

	/*if len(os.Args) < 2 {
		flag.Usage()
		os.Exit(0)
	}*/

	if *OptionVersion {
		fmt.Printf("\n%s Version: %v %v %v %v\n", filepath.Base(os.Args[0]), Version, GitID, GitBranch, BuildDate)
		os.Exit(0)
	}

	//if *OptionFiles {
	FindPacsave(*OptionQuiet)
	//}

	if *OptionD {
		FindOverrideConf()
	}

	//TODO remove /tmp/dconfig/
	os.Remove("/tmp/dconfig/")

}
