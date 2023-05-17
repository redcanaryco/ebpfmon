package ui

import (
	"ebpfmon/utils"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
	"strconv"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var Programs map[int]utils.BpfProgram
var BpftoolPath string
var HavePids bool
var lock sync.Mutex 
var previousPage string
var featureInfo string

type CgroupProgram struct {
	Id int `json:"id"`
	AttachType string `json:"attach_type"`
	AttachFlags string `json:"attach_flags"`
	Name string `json:"name"`
}

type CgroupInfo struct {
	Cgroup string `json:"cgroup"`
	Programs []CgroupProgram `json:"programs"`
}

type XdpInfo struct {
	DevName string `json:"devname"`
	IfIndex int `json:"ifindex"`
	Mode string `json:"mode"`
	Id int `json:"id"`
}

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

type NetInfo struct {
	Xdp []XdpInfo `json:"xdp"`
	Tc []TcInfo `json:"tc"`
	FlowDissector []FlowDissectorInfo `json:"flow_dissector"`
}

type MapContents struct {
	Key []string `json:"key"`
	Value []string `json:"value"`
	Formatted map[string]interface{} `json:"formatted"`
}

type PerfInfo struct {
	Pid int `json:"pid"`
	Fd int `json:"fd"`
	ProgId int `json:"prog_id"`
	FdType string `json:"fd_type"`
	Func string `json:"func",omitempty`
	Offset int `json:"offset",omitempty`
	Filename string `json:"filename",omitempty`
	Tracepoint string `json:"tracepoint",omitempty`
}

func applyNetData() {
	netInfo := []NetInfo{}
	stdout, _, err := utils.RunCmd("sudo", BpftoolPath, "-j", "net", "show")
	if err != nil {
		return 
	}
	err = json.Unmarshal(stdout, &netInfo)
	if err != nil {
		return
	}

	for _, prog := range netInfo {
		for _, xdp := range prog.Xdp {
			if entry, ok := Programs[xdp.Id]; ok {
				entry.Interface = xdp.DevName
				Programs[xdp.Id] = entry
			}
		}
		for _, tc := range prog.Tc {
			// Update programs with tc data
			if entry, ok := Programs[tc.Id]; ok {
				entry.Interface = tc.DevName
				entry.Name = tc.Name
				entry.TcKind = tc.Kind
				Programs[tc.Id] = entry
			}
		}
		// TODO: Add flow dissector data
		// for _, flow := range prog.FlowDissector {
		// 	// Update programs with flow dissector data
		// }
	}
}

// Try an add extra information to the BpfProgram struct
// If it fails that's ok. It just means we won't have the extra info
// This runs as a go routine
func applyCgroupData() {
	cgroupInfo := []CgroupInfo{}
	stdout, _, err := utils.RunCmd("sudo", BpftoolPath, "-j", "cgroup", "tree")
	if err != nil {
		return 
	}
	err = json.Unmarshal(stdout, &cgroupInfo)
	if err != nil {
		return
	}

	for _, prog := range cgroupInfo {
		for _, cgroupProg := range prog.Programs {
			if entry, ok := Programs[cgroupProg.Id]; ok {
				entry.Cgroup = prog.Cgroup
				entry.CgroupAttachFlags = cgroupProg.AttachFlags
				entry.CgroupAttachType = cgroupProg.AttachType
				Programs[cgroupProg.Id] = entry
			}
		}
	}
}

// Adds extra context to the BpfProgram struct
// This runs as a go routine
func enrichPrograms() {
	applyPerfEventData()
	applyCgroupData()
	applyNetData()
	// TODO: Add `bpftool net` output as well
}

// Call the bpftool binary using the `perf` option to get the
// list of perf events
// This runs as a go routine
func applyPerfEventData() {
	perfInfo := []PerfInfo{}
	stdout, stderr, err := utils.RunCmd("sudo", BpftoolPath, "-j", "perf", "list")
	if err != nil {
		fmt.Println("Failed to run `sudo bpftool perf list`")
		fmt.Println(string(stderr))
		panic(err)
	}

	err = json.Unmarshal(stdout, &perfInfo)
	if err != nil {
		panic(err)
	}

	for _, prog := range perfInfo {
		if entry, ok := Programs[prog.ProgId]; ok {
			entry.Fd = prog.Fd
			entry.ProgType = prog.FdType
			if prog.FdType == "kprobe" || prog.FdType == "kretprobe" {
				entry.AttachPoint =  append(entry.AttachPoint, prog.Func)
				entry.Offset = prog.Offset
			} else if prog.FdType == "uprobe" || prog.FdType == "uretprobe" {
				entry.AttachPoint =  append(entry.AttachPoint, prog.Filename)
				entry.Offset = prog.Offset
			} else {
				entry.AttachPoint =  append(entry.AttachPoint, prog.Tracepoint)
			}
			//TODO: Is there a better way to do this? I want me updating entry to just update the item in Programs
			Programs[prog.ProgId] = entry
		}
	}
}

// Call the bpftool binary to gather the list of available programs
// and return a list of BpfProgram structs
// This runs as a go routine and updates the Programs variable
func updateBpfPrograms() {
	// I think the bug is here. We need to intelligently update the Programs variable
	// Use a mutex
	lock.Lock()

	Programs = map[int]utils.BpfProgram{}
	tmp := []utils.BpfProgram{}
	stdout, stderr, err := utils.RunCmd("sudo", BpftoolPath, "-j", "prog", "show")
	if err != nil {
		fmt.Printf("Failed to run `sudo %s -j prog show`\n%s\n", BpftoolPath, string(stderr))
		panic(err)
	}
	err = json.Unmarshal(stdout, &tmp)
	if err != nil {
		panic(err)
	}

	for _, program := range tmp {
		Programs[program.ProgramId] = program
	}

	for _, value := range Programs {
		for j, pid := range value.Pids {
			cmdline, err := utils.GetProcessCmdline(pid.Pid)
			if err == nil {
				value.Pids[j].Cmdline = cmdline
			}
			path, err := utils.GetProcessPath(pid.Pid)
			if err == nil {
				value.Pids[j].Path = path
			}
		}
	}
	enrichPrograms()
	lock.Unlock()
}

// Periodically update stuff
func update(app *tview.Application, list *tview.List) {
	for {
		time.Sleep(3 * time.Second)
		updateBpfPrograms()
		currentSelection := list.GetCurrentItem()

		app.QueueUpdateDraw(func() {
			// Remove all the items
			list.Clear()
			populateList(list)
			list.SetCurrentItem(currentSelection)
		})
	}
}

type Tui struct {
	App *tview.Application
	ProgramList *tview.List
	Disassembly *tview.TextView
	BpfInfoView *tview.TextView
	MapList *tview.List
	MapInfoView *tview.TextView
}

func NewTui(bpftoolPath string) Tui {
	Programs = map[int]utils.BpfProgram{}
	BpftoolPath = bpftoolPath
	disassembly := buildDisassemblyView()
	bpfInfoView := buildBpfInfoView()
	mapInfoView := buildMapInfoView()
	featuresPage := buildFeatureView()
	mapTableView := NewBpfMapTableView()
	app := NewApp()
	pages := tview.NewPages()

	mapList := buildMapList(mapInfoView, mapTableView, pages, app)
	programList := buildProgramList(mapList, bpfInfoView, disassembly, mapInfoView)

	fmt.Println("Collecting bpf information. This may take a few seconds")
	updateBpfPrograms()

	// Set up proper tab navigation and global quit key
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		name, _ := pages.GetFrontPage()
		if name == "programs" {
			if event.Key() == tcell.KeyTab {
				curFocus := app.GetFocus()
				if curFocus == programList {
					app.SetFocus(disassembly)
				} else if curFocus == disassembly {
					app.SetFocus(bpfInfoView)
				} else if curFocus == bpfInfoView {
					app.SetFocus(mapList)
				} else if curFocus == mapList {
					app.SetFocus(mapInfoView)
				} else if curFocus == mapInfoView {
					app.SetFocus(programList)
				}
				return nil
			} else if event.Key() == tcell.KeyBacktab {
				curFocus := app.GetFocus()
				if curFocus == programList {
					app.SetFocus(mapList)
				} else if curFocus == disassembly {
					app.SetFocus(programList)
				} else if curFocus == bpfInfoView {
					app.SetFocus(disassembly)
				} else if curFocus == mapList {
					app.SetFocus(bpfInfoView)
				} else if curFocus == mapInfoView {
					app.SetFocus(mapList)
				}
				return nil
			}
		} else if name == "features" {
			if event.Key() == tcell.KeyTab {
				if featuresPage.GetItem(0).HasFocus() {
					app.SetFocus(featuresPage.GetItem(1))
				} else {
					app.SetFocus(featuresPage.GetItem(0))
				}
				return nil
			} else if event.Key() == tcell.KeyBacktab {
				if featuresPage.GetItem(0).HasFocus() {
					app.SetFocus(featuresPage.GetItem(1))
				} else {
					app.SetFocus(featuresPage.GetItem(0))
				}
				return nil
			}
		}		

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
				featuresPage.GetItem(1).(*tview.TextView).SetText(string(stderr))
			} else {
				featureInfo = string(stdout)
				featuresPage.GetItem(1).(*tview.TextView).SetText(featureInfo)
			}

			// Set focus to the input field
			app.SetFocus(featuresPage.GetItem(0))
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

	// Arrange the UI elements
	frame := buildFrame(programList)
	rightFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(bpfInfoView, 0, 2, false).
		AddItem(mapList, 0, 1, false).
		AddItem(mapInfoView, 0, 2, false)

	// Main flex layout
	// flex := tview.NewFlex().
	// 	AddItem(frame, 0, 1, true).
	// 	AddItem(disassembly, 0, 2, false).
	// 	AddItem(rightFlex, 0, 1, false)
	
	// Alternate layout
	aflex := tview.NewFlex().
		AddItem(disassembly, 0, 2, false).
		AddItem(rightFlex, 0, 1, false)

	flex := tview.NewFlex()
	flex.SetDirection(tview.FlexRow)
	flex.AddItem(frame, 0, 1, true).AddItem(aflex, 0, 2, false)
	
	pages.AddPage("programs", flex, true, true)
	pages.AddPage("help", buildHelpView(), true, false)
	pages.AddPage("features", featuresPage, true, false)
	pages.AddPage("maptable", mapTableView.pages, true, false)
	previousPage = "programs"
	app.SetRoot(pages, true)
	go update(app, programList)
	
	return Tui{
		App: app,
		ProgramList: programList,
		Disassembly: disassembly,
		BpfInfoView: bpfInfoView,
		MapList: mapList,
		MapInfoView: mapInfoView,
	}
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

func buildHelpView() *tview.Modal {
	modal := tview.NewModal()
	modal.SetBorder(true).SetTitle("Help")
	modal.SetText("F1: Help\nCtrl-e: Bpf program view\nCtrl-f: Bpf feature view\n'q'|'Q': Quit")
	return modal
}

func buildProgramList(mapList *tview.List, bpfInfoView *tview.TextView, disassembly *tview.TextView, mapInfoView *tview.TextView) *tview.List {
	list := tview.NewList()
	list.ShowSecondaryText(false)
	populateList(list)

	list.SetSelectedFunc(func(i int, s1, s2 string, r rune) {
		mapList.Clear()
		bpfInfoView.Clear()
		disassembly.Clear()
		mapInfoView.Clear()

		lock.Lock()
		progId, err := strconv.Atoi(strings.TrimSpace(strings.Split(s1, ":")[0]))
		if err != nil {
			fmt.Fprintf(bpfInfoView, "Failed to parse program id: %s\n", err)
		}

		selectedProgram := Programs[progId]

		lock.Unlock()
		insns, err := utils.GetBpfProgramDisassembly(progId)
		if err != nil {
			fmt.Fprintf(disassembly, "Error getting disassembly: %s\n", err)
		} else {
			for _, line := range insns {
				fmt.Fprintf(disassembly, "%s\n", line)
			}
		}


		// Get the map info for each map used by the selected program
		if len(selectedProgram.MapIds) > 0 {
			mapInfo := utils.GetBpfMapInfoByIds(selectedProgram.MapIds)
			for _, map_ := range mapInfo {
				mapList.AddItem(map_.String(), "", 0, nil)
			}
		}

		// Output the info for the selected program
		fmt.Fprintf(bpfInfoView, "[blue]Name:[-] %s\n", selectedProgram.Name)
		fmt.Fprintf(bpfInfoView, "[blue]Tag:[-] %s\n", selectedProgram.Tag)
		fmt.Fprintf(bpfInfoView, "[blue]ProgramId:[-] %d\n", selectedProgram.ProgramId)
		fmt.Fprintf(bpfInfoView, "[blue]ProgType:[-] %s\n", selectedProgram.ProgType)
		for _, pid := range selectedProgram.Pids {
			fmt.Fprintf(bpfInfoView, "[blue]Owner:[-] %s\n", pid.Comm)
			fmt.Fprintf(bpfInfoView, "[blue]OwnerCmdline:[-] %s\n", pid.Cmdline)
			fmt.Fprintf(bpfInfoView, "[blue]OwnerPath:[-] %s\n", pid.Path)
			fmt.Fprintf(bpfInfoView, "[blue]OwnerPid:[-] %d\n", pid.Pid)
			fmt.Fprintf(bpfInfoView, "[blue]OwnerUid:[-] %d\n", pid.Uid)
			fmt.Fprintf(bpfInfoView, "[blue]OwnerGid:[-] %d\n", pid.Gid)
		}
		fmt.Fprintf(bpfInfoView, "[blue]GplCompat:[-] %v\n", selectedProgram.GplCompatible)
		fmt.Fprintf(bpfInfoView, "[blue]LoadedAt:[-] %v\n", time.Unix(int64(selectedProgram.LoadedAt), 0))
		fmt.Fprintf(bpfInfoView, "[blue]BytesXlated:[-] %d\n", selectedProgram.BytesXlated)
		fmt.Fprintf(bpfInfoView, "[blue]Jited:[-] %v\n", selectedProgram.Jited)
		fmt.Fprintf(bpfInfoView, "[blue]BytesMemlock:[-] %d\n", selectedProgram.BytesXlated)
		fmt.Fprintf(bpfInfoView, "[blue]BtfId:[-] %d\n", selectedProgram.BtfId)
		if len(selectedProgram.MapIds) > 0 {
			fmt.Fprintf(bpfInfoView, "[blue]MapIds:[-] %v\n", selectedProgram.MapIds)
		}
		if len(selectedProgram.Pinned) > 0 {
			fmt.Fprintf(bpfInfoView, "[blue]Pinned:[-] %s\n", selectedProgram.Pinned)
		}
		// fmt.Println(selectedProgram.ProgType)
		if selectedProgram.ProgType == "kprobe" ||
		   selectedProgram.ProgType == "kretprobe" ||
		   selectedProgram.ProgType == "tracepoint" ||
		   selectedProgram.ProgType == "raw_tracepoint" ||
		   selectedProgram.ProgType == "uprobe" ||
		   selectedProgram.ProgType == "uretprobe" {
			// fmt.Println(selectedProgram.AttachPoint)
			fmt.Fprintf(bpfInfoView, "[blue]AttachPoint:[-]\n")
			for _, attachPoint := range selectedProgram.AttachPoint {
				fmt.Fprintf(bpfInfoView, "\t└─%s\n", attachPoint)
			}
			fmt.Fprintf(bpfInfoView, "[blue]Offset:[-] %d\n", selectedProgram.Offset)
			fmt.Fprintf(bpfInfoView, "[blue]Fd:[-] %d\n", selectedProgram.Fd)
		}

		if strings.Contains(selectedProgram.ProgType, "xdp") || strings.Contains(selectedProgram.ProgType, "sched") {
			fmt.Fprintf(bpfInfoView, "[blue]Interface:[-] %s\n", selectedProgram.Interface)
		}
		if strings.Contains(selectedProgram.ProgType, "cgroup") {
			fmt.Fprintf(bpfInfoView, "[blue]Cgroup:[-] %s\n", selectedProgram.Cgroup)
			fmt.Fprintf(bpfInfoView, "[blue]CgroupAttachType:[-] %s\n", selectedProgram.CgroupAttachType)
			fmt.Fprintf(bpfInfoView, "[blue]CgroupAttachFlags:[-] %s\n", selectedProgram.CgroupAttachFlags)
		}
	})
	
	return list
}

func buildMapList(mapInfoView *tview.TextView, mapTable *BpfMapTableView, pages *tview.Pages, app *tview.Application) *tview.List {
	mapList := tview.NewList()
	mapList.ShowSecondaryText(false)
	mapList.SetBorder(true).SetTitle("Maps")

	mapList.SetSelectedFunc(func(i int, s1, s2 string, r rune) {
		mapId := strings.TrimSpace(strings.Split(s1, ":")[0])
		mapInfoView.Clear()
		mapInfoView.ScrollToBeginning()
		mapIdInt, _ := strconv.Atoi(mapId)

		// TODO: What is the appropriate way to fail?
		mapInfo := utils.GetBpfMapInfoByIds([]int{mapIdInt})[0]
		mapTable.UpdateMap(mapInfo)

		pages.SwitchToPage("maptable")
	})

	return mapList
}

func buildFrame(programList *tview.List) *tview.Frame {
	frame := tview.NewFrame(programList).AddText("    Id: Type          Tag              Name                 Attach Point", true, tview.AlignLeft, tcell.ColorWhite)
	frame.SetBorder(true).SetTitle("Programs")
	return frame
}

func buildFeatureView() *tview.Flex {
	featureView := tview.NewTextView()
	featureView.SetBorder(true).SetTitle("Features")
	featureView.SetDynamicColors(true)
	form := tview.NewForm().
		AddInputField("Feature", "", 0, nil, func(text string) {
			var filteredText string
			var header string
			var foundHeader bool
			for _, line := range strings.Split(featureInfo, "\n") {
				lineLower := strings.ToLower(line)
				textLower := strings.ToLower(text)
				if strings.HasSuffix(line, ":") || strings.HasSuffix(line, "...") {
					foundHeader = true
					header = "[blue]" + line + "[-]\n"
				} else if strings.Contains(lineLower, textLower) {
					if foundHeader {
						foundHeader = false
						filteredText += header
					}
					index := strings.Index(lineLower, textLower)
					if index != -1 {
						filteredText += line[:index] + "[red]" + line[index:index+len(text)] + "[-]" + line[index+len(text):] + "\n"
					} else {
						filteredText += line + "\n"
					}
				}
			} 
			featureView.SetText(filteredText)
		})
	// form.SetHorizontal(true)
	form.SetBorder(true).SetTitle("Search")

	flex := tview.NewFlex()
	flex.SetDirection(tview.FlexRow)
	flex.AddItem(form, 0, 1, false)
	flex.AddItem(featureView, 0, 4, false)

	return flex
}