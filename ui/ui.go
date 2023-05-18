package ui

import (
	"ebpfmon/utils"
	"fmt"
	"sort"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var Programs map[int]utils.BpfProgram
var BpftoolPath string
var HavePids bool
var lock sync.Mutex 
var previousPage string
var featureInfo string

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
	helpview *HelpView
}

func NewTui(bpftoolPath string) *Tui {
	Programs = map[int]utils.BpfProgram{}
	BpftoolPath = bpftoolPath

	// Initialize the global page manager and the application
	app := NewApp()
	pages := tview.NewPages()
	tui := &Tui{App: app, pages: pages}

	// Create each page object
	tui.bpfExplorerView = NewBpfExplorerView(tui)
	tui.bpfFeatureview = NewBpfFeatureView()
	tui.bpfMapTableView = NewBpfMapTableView()
	tui.helpview = NewHelpView()

	fmt.Println("Collecting bpf information. This may take a few seconds")
	updateBpfPrograms()

	// Set up proper tab navigation and global quit key
	// TODO: We may be able to move this stuff to be local to each page instead of at the app level
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		name, _ := pages.GetFrontPage()
		if name == "programs" {
			if event.Key() == tcell.KeyTab {
				curFocus := app.GetFocus()
				if curFocus == tui.bpfExplorerView.programList {
					app.SetFocus(tui.bpfExplorerView.disassembly)
				} else if curFocus == tui.bpfExplorerView.disassembly {
					app.SetFocus(tui.bpfExplorerView.bpfInfoView)
				} else if curFocus == tui.bpfExplorerView.bpfInfoView {
					app.SetFocus(tui.bpfExplorerView.mapList)
				} else if curFocus == tui.bpfExplorerView.mapList {
					app.SetFocus(tui.bpfExplorerView.programList)
				}
				return nil
			} else if event.Key() == tcell.KeyBacktab {
				curFocus := app.GetFocus()
				if curFocus == tui.bpfExplorerView.programList {
					app.SetFocus(tui.bpfExplorerView.mapList)
				} else if curFocus == tui.bpfExplorerView.disassembly {
					app.SetFocus(tui.bpfExplorerView.programList)
				} else if curFocus == tui.bpfExplorerView.bpfInfoView {
					app.SetFocus(tui.bpfExplorerView.disassembly)
				} else if curFocus == tui.bpfExplorerView.mapList {
					app.SetFocus(tui.bpfExplorerView.bpfInfoView)
				}
				return nil
			}
		} else if name == "features" {
			if event.Key() == tcell.KeyTab {
				if tui.bpfFeatureview.flex.GetItem(0).HasFocus() {
					app.SetFocus(tui.bpfFeatureview.flex.GetItem(1))
				} else {
					app.SetFocus(tui.bpfFeatureview.flex.GetItem(0))
				}
				return nil
			} else if event.Key() == tcell.KeyBacktab {
				if tui.bpfFeatureview.flex.GetItem(0).HasFocus() {
					app.SetFocus(tui.bpfFeatureview.flex.GetItem(1))
				} else {
					app.SetFocus(tui.bpfFeatureview.flex.GetItem(0))
				}
				return nil
			}
		}		

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
			
			// Run bpftool feature command and display the output (or stderr on failure)
			stdout, stderr, err := utils.RunCmd("sudo", BpftoolPath, "feature", "probe")
			if err != nil {
				tui.bpfFeatureview.flex.GetItem(1).(*tview.TextView).SetText(string(stderr))
			} else {
				featureInfo = string(stdout)
				tui.bpfFeatureview.flex.GetItem(1).(*tview.TextView).SetText(featureInfo)
			}

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
		} 
		return event
	})
	
	pages.AddPage("programs", tui.bpfExplorerView.flex, true, true)
	pages.AddPage("help", tui.helpview.modal, true, false)
	pages.AddPage("features", tui.bpfFeatureview.flex, true, false)
	pages.AddPage("maptable", tui.bpfMapTableView.pages, true, false)
	previousPage = "programs"
	app.SetRoot(pages, true)
	tui.bpfExplorerView.Update(tui)
	return tui
}

func NewApp() *tview.Application {
	app := tview.NewApplication()
	return app
}

// Populate a tview.List with the output of GetBpfPrograms
func populateList(list *tview.List)  {
	keys := make([]int, 0, len(Programs))
    for k := range Programs{
        keys = append(keys, k)
    }
    sort.Ints(keys)
 
    for _, k := range keys {
		list.AddItem(Programs[k].String(), "", 0, nil)
    }
}