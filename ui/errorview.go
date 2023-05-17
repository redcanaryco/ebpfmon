// This page is a simple help modal for showing what the keys are for navigating
package ui

import "github.com/rivo/tview"

type ErrorView struct {
	modal *tview.Modal
}

func NewErrorView() *ErrorView {
	e := &ErrorView{}
	e.buildErrorView()
	return e
}

func (e *ErrorView) buildErrorView() {
	modal := tview.NewModal()
	modal.SetBorder(true).SetTitle("Error")
	modal.SetText("F1: Help\nCtrl-e: Bpf program view\nCtrl-f: Bpf feature view\n'q'|'Q': Quit")
	e.modal = modal
}

func (e *ErrorView) SetError(err string) {
	e.modal.SetText(err)
}
