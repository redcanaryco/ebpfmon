// Description: Main file for the ebpfmon tool
// Author: Dave Bogle
//
// This tool is used to visualize the bpf programs and maps that are loaded
// on a system. It uses the bpftool binary to get the information about the
// programs and maps and then displays them in a tui using the tview library.
// The user can select a program and then see the maps that are used by that
// program. The user can also see the disassembly of a program by selecting
// the program using the enter key
package main

import (
	"ebpfmon/ui"
	"ebpfmon/utils"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)


var BpftoolPath string
var HavePids bool

type BpftoolVersionInfo struct {
	Version string `json:"version"`
	LibbpfVersion string `json:"libbpf_version"`
	Features struct {
		Libbfd bool `json:"libbfd"`
		Llvm bool `json:"llvm"`
		Skeletons bool `json:"skeletons"`
		Bootstrap bool `json:"bootstrap"`
	} `json:"features"`
}

type Config struct {
	Version BpftoolVersionInfo
	BpftoolPath string
	HavePids bool
	Verbose bool
}

func main() {
	// Parse the command line arguments
	var err error
	help := flag.Bool("help", false, "Display help")
	verbose := flag.Bool("verbose", false, "Verbose output")
	version := flag.Bool("version", false, "Display version information")
	bpftool_path := flag.String("bpftool", "", "Path to bpftool binary. Defaults to the bpftool located in PATH")

	flag.Parse()

	if *help {
		fmt.Println("ebpfmon is a tool for monitoring bpf programs")
		flag.Usage()
		return
	}

	if *verbose {
		// TODO: Set verbose output
	}

	if *version {
		fmt.Println("bpfmon version 0.1")
		return
	}

	// Set the global bpftool path variable. It can be set by the command line
	// argument or by the BPFTOOL_PATH environment variable. It defaults to
	// the bpftool binary in the PATH
	bpftoolEnvPath, exists := os.LookupEnv("BPFTOOL_PATH")
	if *bpftool_path != "" {
		_, err := os.Stat(*bpftool_path)
		if err != nil {
			fmt.Printf("Failed to find bpftool binary at %s\n", *bpftool_path)
			return
		}
		BpftoolPath = *bpftool_path
	} else if exists {
		_, err := os.Stat(bpftoolEnvPath)
		if err != nil {
			fmt.Printf("Failed to find bpftool binary specified by BPFTOOL_PATH at %s\n", bpftoolEnvPath)
			return
		}
		BpftoolPath = bpftoolEnvPath
	} else {
		BpftoolPath, err = filepath.Abs("./.output/bpftool")
		if err != nil {
			fmt.Println("Failed to find compiled version of bpftool")
			return
		}

	}

	versionInfo := BpftoolVersionInfo{}
	stdout, stderr, err := utils.RunCmd(BpftoolPath, "version", "-j")
	if err != nil {
		fmt.Printf("Failed to run `%s version -j`\n%s\n", BpftoolPath, string(stderr))
		return
	}
	err = json.Unmarshal(stdout, &versionInfo)
	if err != nil {
		fmt.Println("Failed to parse bpftool version output")
		return
	}
	if versionInfo.Features.Skeletons {
		HavePids = true
	}

	config := Config {
		Version: versionInfo,
		BpftoolPath: BpftoolPath,
		HavePids: HavePids,
		Verbose: *verbose,
	}
	utils.BpftoolPath = config.BpftoolPath
	app := ui.NewTui(config.BpftoolPath)

	// Run the app
	if err := app.App.Run(); err != nil {
		panic(err)
	}
}