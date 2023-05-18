// This file handles the features page of the TUI. It is used to display the
// features that are supported by the kernel and the bpftool binary. It also
// allows the user to search the features to find the ones they are interested
// in
package ui

import (
	"ebpfmon/utils"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type BpfFeatureView struct {
	flex *tview.Flex
}

func NewBpfFeatureView(tui *Tui) *BpfFeatureView {
	result := &BpfFeatureView{}
	result.buildFeatureView(tui)
	return result
}

func (b *BpfFeatureView) buildFeatureView(tui *Tui) {
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
	flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			if flex.GetItem(0).HasFocus() {
				tui.App.SetFocus(flex.GetItem(1))
			} else {
				tui.App.SetFocus(flex.GetItem(0))
			}
			return nil
		} else if event.Key() == tcell.KeyBacktab {
			if flex.GetItem(0).HasFocus() {
				tui.App.SetFocus(flex.GetItem(1))
			} else {
				tui.App.SetFocus(flex.GetItem(0))
			}
			return nil
		}
		return event
	})
	
	// Run bpftool feature command and display the output (or stderr on failure)
	stdout, stderr, err := utils.RunCmd("sudo", BpftoolPath, "feature", "probe")
	if err != nil {
		flex.GetItem(1).(*tview.TextView).SetText(string(stderr))
	} else {
		featureInfo = string(stdout)
		flex.GetItem(1).(*tview.TextView).SetText(featureInfo)
	}

	b.flex = flex
}