package ui

import (
	"strings"

	"github.com/rivo/tview"
)

type BpfFeatureView struct {
	flex *tview.Flex
}

func NewBpfFeatureView() *BpfFeatureView {
	result := &BpfFeatureView{}
	result.buildFeatureView()
	return result
}

func (b *BpfFeatureView) buildFeatureView() {
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
	b.flex = flex
}