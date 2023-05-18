// This is the file for managing the main bpf explorer page. This page is used
// to display all the bpf programs that are loaded on the system. It also
// displays the maps that are used by each program. The user can select a
// program and then see the disassembly of that program. The user can also
// see the maps that are used by the program. Those maps can be selected by the
// user which will display the map view/edit page
package ui

import (
	"ebpfmon/utils"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

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

type NetInfo struct {
	Xdp []XdpInfo `json:"xdp"`
	Tc []TcInfo `json:"tc"`
	FlowDissector []FlowDissectorInfo `json:"flow_dissector"`
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

type BpfExplorerView struct {
	flex *tview.Flex
	programList *tview.List
	disassembly *tview.TextView
	bpfInfoView *tview.TextView
	mapList *tview.List
}

func applyNetData() {
	netInfo := []NetInfo{}
	stdout, _, err := utils.RunCmd("sudo", BpftoolPath, "-j", "net", "show")
	if err != nil {
		logger.Printf("Error running `sudo %s -j net show`: %s\n", BpftoolPath, err)
	}
	err = json.Unmarshal(stdout, &netInfo)
	if err != nil {
		logger.Printf("Error decoding json output of `sudo %s -j net show`: %s\n", BpftoolPath, err)
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
		logger.Printf("Error running `sudo %s -j cgroup tree`: %s\n", BpftoolPath, err)
	}
	err = json.Unmarshal(stdout, &cgroupInfo)
	if err != nil {
		logger.Printf("Error decoding json output of `sudo %s -j cgroup tree`: %s\n", BpftoolPath, err)
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
	stdout, _, err := utils.RunCmd("sudo", BpftoolPath, "-j", "perf", "list")
	if err != nil {
		logger.Printf("Error running `sudo %s -j perf list`: %s\n", BpftoolPath, err)
	}

	err = json.Unmarshal(stdout, &perfInfo)
	if err != nil {
		logger.Printf("Error decoding json output of `sudo %s -j perf list`: %s\n", BpftoolPath, err)
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
func updateBpfPrograms(tui *Tui) {
	// I think the bug is here. We need to intelligently update the Programs variable
	// Use a mutex
	lock.Lock()

	Programs = map[int]utils.BpfProgram{}
	tmp := []utils.BpfProgram{}
	stdout, stderr, err := utils.RunCmd("sudo", BpftoolPath, "-j", "prog", "show")
	if err != nil {
		tui.DisplayError(fmt.Sprintf("Failed to run `sudo %s -j prog show`\n%s\n", BpftoolPath, string(stderr)))
	}
	err = json.Unmarshal(stdout, &tmp)
	if err != nil {
		tui.DisplayError(fmt.Sprintf("Failed to run `sudo %s -j prog show`\n%s\n", BpftoolPath, string(stderr)))
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

func NewBpfExplorerView(tui *Tui) *BpfExplorerView {
	BpfExplorerView := &BpfExplorerView{}
	BpfExplorerView.buildProgramList()
	BpfExplorerView.buildMapList(tui)
	BpfExplorerView.buildDisassemblyView()
	BpfExplorerView.buildBpfInfoView()
	BpfExplorerView.buildLayout(tui)
	return BpfExplorerView
}

func (b* BpfExplorerView) Update(tui *Tui) {
	for {
		time.Sleep(3 * time.Second)
		updateBpfPrograms(tui)
		currentSelection := b.programList.GetCurrentItem()

		tui.App.QueueUpdateDraw(func() {
			// Remove all the items
			b.programList.Clear()
			populateList(b.programList)
			b.programList.SetCurrentItem(currentSelection)
		})
	}
}

func (b *BpfExplorerView) buildLayout(tui *Tui) {
	// Arrange the UI elements
	frame := buildFrame(b.programList)
	rightFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(b.bpfInfoView, 0, 2, false).
		AddItem(b.mapList, 0, 1, false)
	
	// Alternate layout
	aflex := tview.NewFlex().
		AddItem(b.disassembly, 0, 2, false).
		AddItem(rightFlex, 0, 1, false)

	b.flex = tview.NewFlex()
	b.flex.SetDirection(tview.FlexRow)
	b.flex.AddItem(frame, 0, 1, true).AddItem(aflex, 0, 2, false)
	b.flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			curFocus := tui.App.GetFocus()
			if curFocus == b.programList {
				tui.App.SetFocus(b.disassembly)
			} else if curFocus == b.disassembly {
				tui.App.SetFocus(b.bpfInfoView)
			} else if curFocus == b.bpfInfoView {
				tui.App.SetFocus(b.mapList)
			} else if curFocus == b.mapList {
				tui.App.SetFocus(b.programList)
			}
			return nil
		} else if event.Key() == tcell.KeyBacktab {
			curFocus := tui.App.GetFocus()
			if curFocus == b.programList {
				tui.App.SetFocus(b.mapList)
			} else if curFocus == b.disassembly {
				tui.App.SetFocus(b.programList)
			} else if curFocus == b.bpfInfoView {
				tui.App.SetFocus(b.disassembly)
			} else if curFocus == b.mapList {
				tui.App.SetFocus(b.bpfInfoView)
			}
			return nil
		}
		return event
	})
}

func (b *BpfExplorerView) buildProgramList() {
	b.programList = tview.NewList()
	b.programList.ShowSecondaryText(false)
	populateList(b.programList)

	b.programList.SetSelectedFunc(func(i int, s1, s2 string, r rune) {
		b.mapList.Clear()
		b.bpfInfoView.Clear()
		b.disassembly.Clear()

		lock.Lock()
		progId, err := strconv.Atoi(strings.TrimSpace(strings.Split(s1, ":")[0]))
		if err != nil {
			fmt.Fprintf(b.bpfInfoView, "Failed to parse program id: %s\n", err)
		}

		selectedProgram := Programs[progId]

		lock.Unlock()
		insns, err := utils.GetBpfProgramDisassembly(progId)
		if err != nil {
			fmt.Fprintf(b.disassembly, "Error getting disassembly: %s\n", err)
		} else {
			for _, line := range insns {
				fmt.Fprintf(b.disassembly, "%s\n", line)
			}
		}


		// Get the map info for each map used by the selected program
		if len(selectedProgram.MapIds) > 0 {
			mapInfo, err := utils.GetBpfMapInfoByIds(selectedProgram.MapIds)
			if err != nil {
				fmt.Fprintf(b.bpfInfoView, "Failed to get map info: %s\n", err)
			}
			for _, map_ := range mapInfo {
				b.mapList.AddItem(map_.String(), "", 0, nil)
			}
		}

		// Output the info for the selected program
		fmt.Fprintf(b.bpfInfoView, "[blue]Name:[-] %s\n", selectedProgram.Name)
		fmt.Fprintf(b.bpfInfoView, "[blue]Tag:[-] %s\n", selectedProgram.Tag)
		fmt.Fprintf(b.bpfInfoView, "[blue]ProgramId:[-] %d\n", selectedProgram.ProgramId)
		fmt.Fprintf(b.bpfInfoView, "[blue]ProgType:[-] %s\n", selectedProgram.ProgType)
		for _, pid := range selectedProgram.Pids {
			fmt.Fprintf(b.bpfInfoView, "[blue]Owner:[-] %s\n", pid.Comm)
			fmt.Fprintf(b.bpfInfoView, "[blue]OwnerCmdline:[-] %s\n", pid.Cmdline)
			fmt.Fprintf(b.bpfInfoView, "[blue]OwnerPath:[-] %s\n", pid.Path)
			fmt.Fprintf(b.bpfInfoView, "[blue]OwnerPid:[-] %d\n", pid.Pid)
			fmt.Fprintf(b.bpfInfoView, "[blue]OwnerUid:[-] %d\n", pid.Uid)
			fmt.Fprintf(b.bpfInfoView, "[blue]OwnerGid:[-] %d\n", pid.Gid)
		}
		fmt.Fprintf(b.bpfInfoView, "[blue]GplCompat:[-] %v\n", selectedProgram.GplCompatible)
		fmt.Fprintf(b.bpfInfoView, "[blue]LoadedAt:[-] %v\n", time.Unix(int64(selectedProgram.LoadedAt), 0))
		fmt.Fprintf(b.bpfInfoView, "[blue]BytesXlated:[-] %d\n", selectedProgram.BytesXlated)
		fmt.Fprintf(b.bpfInfoView, "[blue]Jited:[-] %v\n", selectedProgram.Jited)
		fmt.Fprintf(b.bpfInfoView, "[blue]BytesMemlock:[-] %d\n", selectedProgram.BytesXlated)
		fmt.Fprintf(b.bpfInfoView, "[blue]BtfId:[-] %d\n", selectedProgram.BtfId)
		if len(selectedProgram.MapIds) > 0 {
			fmt.Fprintf(b.bpfInfoView, "[blue]MapIds:[-] %v\n", selectedProgram.MapIds)
		}
		if len(selectedProgram.Pinned) > 0 {
			fmt.Fprintf(b.bpfInfoView, "[blue]Pinned:[-] %s\n", selectedProgram.Pinned)
		}
		// fmt.Println(selectedProgram.ProgType)
		if selectedProgram.ProgType == "kprobe" ||
		   selectedProgram.ProgType == "kretprobe" ||
		   selectedProgram.ProgType == "tracepoint" ||
		   selectedProgram.ProgType == "raw_tracepoint" ||
		   selectedProgram.ProgType == "uprobe" ||
		   selectedProgram.ProgType == "uretprobe" {
			// fmt.Println(selectedProgram.AttachPoint)
			fmt.Fprintf(b.bpfInfoView, "[blue]AttachPoint:[-]\n")
			for _, attachPoint := range selectedProgram.AttachPoint {
				fmt.Fprintf(b.bpfInfoView, "\t└─%s\n", attachPoint)
			}
			fmt.Fprintf(b.bpfInfoView, "[blue]Offset:[-] %d\n", selectedProgram.Offset)
			fmt.Fprintf(b.bpfInfoView, "[blue]Fd:[-] %d\n", selectedProgram.Fd)
		}

		if strings.Contains(selectedProgram.ProgType, "xdp") || strings.Contains(selectedProgram.ProgType, "sched") {
			fmt.Fprintf(b.bpfInfoView, "[blue]Interface:[-] %s\n", selectedProgram.Interface)
		}
		if strings.Contains(selectedProgram.ProgType, "cgroup") {
			fmt.Fprintf(b.bpfInfoView, "[blue]Cgroup:[-] %s\n", selectedProgram.Cgroup)
			fmt.Fprintf(b.bpfInfoView, "[blue]CgroupAttachType:[-] %s\n", selectedProgram.CgroupAttachType)
			fmt.Fprintf(b.bpfInfoView, "[blue]CgroupAttachFlags:[-] %s\n", selectedProgram.CgroupAttachFlags)
		}
	})
}

func (b *BpfExplorerView) buildMapList(tui *Tui){
	b.mapList = tview.NewList()
	b.mapList.ShowSecondaryText(false)
	b.mapList.SetBorder(true).SetTitle("Maps")

	b.mapList.SetSelectedFunc(func(i int, s1, s2 string, r rune) {
		mapId := strings.TrimSpace(strings.Split(s1, ":")[0])
		mapIdInt, _ := strconv.Atoi(mapId)

		// TODO: What is the appropriate way to fail?
		mapInfo, err := utils.GetBpfMapInfoByIds([]int{mapIdInt})
		if err != nil {
			logger.Printf("Failed to get map info: %s\n", err)
		}
		tui.bpfMapTableView.UpdateMap(mapInfo[0])

		tui.pages.SwitchToPage("maptable")
	})
}

func buildFrame(programList *tview.List) *tview.Frame {
	frame := tview.NewFrame(programList).AddText("    Id: Type          Tag              Name                 Attach Point", true, tview.AlignLeft, tcell.ColorWhite)
	frame.SetBorder(true).SetTitle("Programs")
	return frame
}

func (b *BpfExplorerView) buildBpfInfoView() {
	b.bpfInfoView = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true)
	b.bpfInfoView.SetBorder(true).SetTitle("Info")
}

func (b *BpfExplorerView) buildDisassemblyView() {
	b.disassembly = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true)
	b.disassembly.SetBorder(true).SetTitle("Disassembly")
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