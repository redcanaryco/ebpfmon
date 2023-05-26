// This page is a simple log view for showing what logs a given program has
package ui

import (
	"ebpfmon/utils"
	"errors"
	"fmt"
	"os"
	"strings"
	// "syscall"

	"github.com/rivo/tview"
	log "github.com/sirupsen/logrus"
)

type LogView struct {
	view *tview.TextView
	tui *Tui
}


func NewLogView(t *Tui) *LogView {
	e := &LogView{tui: t}
	e.buildLogView()
	return e
}

func validateTracePipe(tracePipePath string) bool {
	stdout, _, err := utils.RunCmd("sudo", "stat", "-f", tracePipePath)
	if err != nil {
		log.Warningf("Failed to statfs trace pipe: %s", tracePipePath)
		log.Warning(err)
		return false
	}

	log.Info(string(stdout))
	if strings.Contains(string(stdout), "Type: tracefs") {
		log.Info("statfs succeeded in finding trace_pipe")
		return true
	}
	return false
	// info := syscall.Statfs_t{}
	// err := syscall.Statfs(tracePipePath, &info)
	// if err != nil {
	// 	log.Warningf("Failed to statfs trace pipe: %s", tracePipePath)
	// 	log.Warning(err)
	// 	return false
	// }
	// log.Info(info.Type)
	// if info.Type == 0x74726163 {
	// 	return true
	// }
	// return false
}

func findTracePipe() (string, error) {
	possibleMnts := []string{
		"/sys/kernel/debug/tracing",
		"/sys/kernel/tracing",
		"/tracing",
		"/trace",
	};

	for _, mnt := range possibleMnts {
		if validateTracePipe(mnt + "/trace_pipe") {
			log.Debugf("Found trace pipe at %s", mnt)
			return mnt + "/trace_pipe", nil
		}
	}
	return "", errors.New("Could not find trace pipe")
}

func monitorTracePipe(tracePipePath string, e *LogView) {
	
	f, err := os.Open(tracePipePath)
	if err != nil {
		tui.DisplayError("Failed to open trace pipe")
		return
	}
	defer f.Close()

	for {
		buf := make([]byte, 1024)
		n, err := f.Read(buf)
		if err != nil {
			tui.DisplayError("Failed to read trace pipe")
			return
		}
		if n > 0 {
			fmt.Fprintf(e.view, "%s", (buf[:n]))
		}
	}
}

func (e *LogView) StartMonitor() {
	tracePipePath, err := findTracePipe()
	if err != nil {
		tui.DisplayError("Failed to find trace pipe path")
		return
	}

	go monitorTracePipe(tracePipePath, e)
}

func (e *LogView) buildLogView() {
	view := tview.NewTextView()
	view.SetBorder(true).SetTitle("Log")
	e.view = view
}
