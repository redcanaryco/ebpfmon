package ui

import "github.com/rivo/tview"

func buildMapInfoView() *tview.TextView {
	mapInfoView := tview.NewTextView()
	mapInfoView.SetScrollable(true).SetTitle("Map Info").SetBorder(true)
	return mapInfoView
}

func buildBpfInfoView() *tview.TextView {
	bpfInfoView := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true)
	bpfInfoView.SetBorder(true).SetTitle("Info")
	return bpfInfoView
}

func buildDisassemblyView() *tview.TextView {
	disassembly := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true)
	disassembly.SetBorder(true).SetTitle("Disassembly")
	return disassembly
}