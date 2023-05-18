// This is the core code for building the TUI application portion of ebpfmon
// This page handles building the root application and the pages that are used
// to display each of the view the app supports. It also handles the global
// keybindings and the global state of the application
package ui

import (
	"ebpfmon/utils"
	"fmt"
	"sync"
	"log"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var Programs map[int]utils.BpfProgram
var BpftoolPath string
var lock sync.Mutex 
var previousPage string
var featureInfo string
var logger *log.Logger

type FlowDissectorInfo struct {
	DevName string `json:"devname"`
	IfIndex int `json:"ifindex"`
	Id int `json:"id"`
}

type TcInfo struct {
	DevName string `json:"devname"`
	IfIndex int `json:"ifindex"`
	Kind string `json:"kind"`
	Name string `json:"name"`
	Id int `json:"id"`
}

type Tui struct {
	App *tview.Application
	pages *tview.Pages
	bpfExplorerView *BpfExplorerView
	bpfMapTableView *BpfMapTableView
	bpfFeatureview *BpfFeatureView
	helpView *HelpView
	errorView *ErrorView
}

func (t *Tui) DisplayError(err string) {
	t.errorView.SetError(err)
	previousPage, _ = t.pages.GetFrontPage()
	t.pages.SwitchToPage("error")
}

func NewTui(bpftoolPath string, l *log.Logger) *Tui {
	logger = l
	Programs = map[int]utils.BpfProgram{}
	BpftoolPath = bpftoolPath

	// Initialize the global page manager and the application
	app := NewApp()
	pages := tview.NewPages()
	tui := &Tui{App: app, pages: pages}

	// Create each page object
	tui.bpfExplorerView = NewBpfExplorerView(tui)
	tui.bpfFeatureview = NewBpfFeatureView(tui)
	tui.bpfMapTableView = NewBpfMapTableView()
	tui.helpView = NewHelpView()
	tui.errorView = NewErrorView()

	fmt.Println("Collecting bpf information. This may take a few seconds")
	updateBpfPrograms(tui)

	// Set up proper page navigation and global quit key
	// In page navigation happens in their respective files
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {	
		// Set up q quit key and page navigation
		if event.Rune() == 'q' || event.Rune() == 'Q' {
			app.Stop()
			return nil
		} else if event.Key() == tcell.KeyCtrlE {
			page, _ := pages.GetFrontPage()
			if page != "help" {
				previousPage = page
			}
			pages.SwitchToPage("programs")
			_, prim := pages.GetFrontPage()
			app.SetFocus(prim)
			return nil
		} else if event.Key() == tcell.KeyCtrlF {
			page, _ := pages.GetFrontPage()
			if page != "help" {
				previousPage = page
			}

			pages.SwitchToPage("features")

			// Set focus to the input field
			app.SetFocus(tui.bpfFeatureview.flex.GetItem(0))
			return nil
		}  else if event.Key() == tcell.KeyF1 || event.Rune() == '?' {
			name, _ := pages.GetFrontPage()
			if name == "help" {
				pages.SwitchToPage(previousPage)
			} else {
				page, _ := pages.GetFrontPage()
				if page != "help" {
					previousPage = page
				}
				pages.SwitchToPage("help")
				_, prim := pages.GetFrontPage()
				app.SetFocus(prim)
			}
			return nil
		} else if event.Key() == tcell.KeyESC {
			pages.SwitchToPage(previousPage)
			_, prim := pages.GetFrontPage()
			app.SetFocus(prim)
			return nil
		}
		return event
	})
	
	// These are the main pages for the application
	pages.AddPage("programs", tui.bpfExplorerView.flex, true, true)
	pages.AddPage("help", tui.helpView.modal, true, false)
	pages.AddPage("features", tui.bpfFeatureview.flex, true, false)
	pages.AddPage("maptable", tui.bpfMapTableView.pages, true, false)
	pages.AddPage("error", tui.errorView.modal, true, false)

	// Set starting page as previous page
	previousPage = "programs"

	// Set the page view as the root
	app.SetRoot(pages, true)

	// Start the go routine to update bpf programs and maps
	go tui.bpfExplorerView.Update(tui)

	return tui
}

func NewApp() *tview.Application {
	app := tview.NewApplication()
	return app
}