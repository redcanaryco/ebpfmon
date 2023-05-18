package ui

import "github.com/rivo/tview"

type HelpView struct {
	modal *tview.Modal
}

func NewHelpView() *HelpView {
	v := &HelpView{}
	v.buildHelpView()
	return v
}	

func (h *HelpView) buildHelpView() {
	modal := tview.NewModal()
	modal.SetBorder(true).SetTitle("Help")
	modal.SetText("F1: Help\nCtrl-e: Bpf program view\nCtrl-f: Bpf feature view\n'q'|'Q': Quit")
	h.modal = modal
}