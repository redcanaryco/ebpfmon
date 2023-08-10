// Description: Main file for the ebpfmon tool
// Author: research@redcanary.com
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
	"os/exec"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

// The global path to the bpftool binary
var BpftoolPath string

// A struct for storing the output of `bpftool version -j`
type BpftoolVersionInfo struct {
	Version       string `json:"version"`
	LibbpfVersion string `json:"libbpf_version"`
	Features      struct {
		Libbfd    bool `json:"libbfd"`
		Llvm      bool `json:"llvm"`
		Skeletons bool `json:"skeletons"`
		Bootstrap bool `json:"bootstrap"`
	} `json:"features"`
}

// A simple struct for storing the config for the app
type Config struct {
	// The version of bpftool that is being used
	Version BpftoolVersionInfo

	// The path to the bpftool binary
	BpftoolPath string

	// Logging verbosity
	Verbose bool
}

func main() {
	// Parse the command line arguments
	var err error
	help := flag.Bool("help", false, "Display help")
	verbose := flag.Bool("verbose", false, "Verbose output")
	version := flag.Bool("version", false, "Display version information")
	logFileArg := flag.String("logfile", "", "Path to log file. Defaults to log.txt")
	bpftool_path := flag.String("bpftool", "", "Path to bpftool binary. Defaults to the bpftool located in PATH")

	flag.Parse()

	if *help {
		fmt.Println("ebpfmon is a tool for monitoring bpf programs")
		flag.Usage()
		return
	}

	if *version {
		fmt.Println("ebpfmon version 0.1")
		return
	}

	var logFile *os.File
	var logpath string
	if *logFileArg == "" {
		logpath, err = filepath.Abs("./log.txt")
		if err != nil {
			fmt.Println("Failed to find log file")
			os.Exit(1)
		}
	} else {
		logpath, err = filepath.Abs(*logFileArg)
		if err != nil {
			fmt.Println("Failed to find log file")
			os.Exit(1)
		}
	}
	logFile, err = os.OpenFile(logpath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Failed to open log file %s\n%v", logpath, err)
		os.Exit(1)
	}
	defer logFile.Close()
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(logFile)
	if *verbose {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	// Set the global bpftool path variable. It can be set by the command line
	// argument or by the BPFTOOL_PATH environment variable. It defaults to
	// the bpftool binary in the PATH
	bpftoolEnvPath, exists := os.LookupEnv("BPFTOOL_PATH")
	if *bpftool_path != "" {
		_, err := os.Stat(*bpftool_path)
		if err != nil {
			fmt.Printf("Failed to find bpftool binary at %s\n", *bpftool_path)
			os.Exit(1)
		}
		BpftoolPath = *bpftool_path
	} else if exists {
		_, err := os.Stat(bpftoolEnvPath)
		if err != nil {
			fmt.Printf("Failed to find bpftool binary specified by BPFTOOL_PATH at %s\n", bpftoolEnvPath)
			os.Exit(1)
		}
		BpftoolPath = bpftoolEnvPath
	} else {
		BpftoolPath, err = exec.LookPath("bpftool")
		if err != nil {
			fmt.Println("Failed to find compiled version of bpftool")
			os.Exit(1)
		} else {
			BpftoolPath, err = filepath.Abs(BpftoolPath)
			if err != nil {
				fmt.Println("Failed to find compiled version of bpftool")
				os.Exit(1)
			}
		}
	}

	versionInfo := BpftoolVersionInfo{}
	stdout, stderr, err := utils.RunCmd(BpftoolPath, "version", "-j")
	if err != nil {
		fmt.Printf("Failed to run `%s version -j`\n%s\n", BpftoolPath, string(stderr))
		os.Exit(1)
	}
	err = json.Unmarshal(stdout, &versionInfo)
	if err != nil {
		fmt.Println("Failed to parse bpftool version output")
		os.Exit(1)
	}

	config := Config{
		Version:     versionInfo,
		BpftoolPath: BpftoolPath,
		Verbose:     *verbose,
	}
	utils.BpftoolPath = config.BpftoolPath
	app := ui.NewTui(config.BpftoolPath)
	log.Info("Starting ebpfmon")

	// Run the app
	if err := app.App.Run(); err != nil {
		panic(err)
	}
}
