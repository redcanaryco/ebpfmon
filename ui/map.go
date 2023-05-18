// This page handles the the view for looking at the map entries of a map. It
// allows the user to select a map and then view the entries in that map. The
// user can also edit the map entries from this view.
package ui

import (
	"ebpfmon/utils"
	"fmt"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type BpfMapTableView struct {
	pages *tview.Pages
	form *tview.Form
	table *tview.Table
	confirm *tview.Modal
	Map utils.BpfMap
	MapEntries []utils.BpfMapEntry
}

// Update the table view with the new map entries
func (b *BpfMapTableView) updateTable() {
	b.table.Clear()
	b.table.SetCell(0, 0, tview.NewTableCell("Index").SetSelectable(false))
	b.table.SetCell(0, 1, tview.NewTableCell("Key").SetSelectable(false))
	b.table.SetCell(0, 2, tview.NewTableCell("Value").SetSelectable(false))
	b.table.SetCell(0, 3, tview.NewTableCell("Formatted").SetSelectable(false))
	for i, entry := range b.MapEntries {
		b.table.SetCell(i+1, 0, tview.NewTableCell(strconv.Itoa(i)))
		b.table.SetCell(i+1, 1, tview.NewTableCell(fmt.Sprintf("%v", entry.Key)))
		b.table.SetCell(i+1, 2, tview.NewTableCell(fmt.Sprintf("%v", entry.Value)))
		b.table.SetCell(i+1, 3, tview.NewTableCell(fmt.Sprintf("%v", entry.Formatted.Value)))
	}
}

// Update Map
func (b *BpfMapTableView) UpdateMap(m utils.BpfMap) {
	var err error
	b.Map = m
	b.MapEntries, err = utils.GetBpfMapEntries(b.Map.Id)
	if err != nil {
		logger.Printf("Error getting map entries: %v\n", err)
	} else {
		b.updateTable()
	}
}

func (b *BpfMapTableView) buildMapTableView() {
	b.table.SetBorder(true).SetTitle("Map Info")
	b.table.SetSelectable(true, false)
	b.table.Select(1, 0)
	b.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {

		// If the user presses the esc key they should go back to the main view
		if event.Key() == tcell.KeyEsc {
			// app.SetFocus("main")
			return nil
		} else if event.Rune() == 'd' {
			b.pages.SwitchToPage("confirm")
			return nil
		}
		return event
	})
	b.table.SetSelectedFunc(func(row int, column int) {
		key := b.table.GetCell(row, 1)
		value := b.table.GetCell(row, 2)
		b.form.GetFormItemByLabel("Key").(*tview.InputField).SetText(key.Text)
		b.form.GetFormItemByLabel("Value").(*tview.InputField).SetText(value.Text)
		b.pages.SwitchToPage("form")
	})
}

func cellTextToHexString(cellValue string) string {
	return strings.Trim(cellValue, "[]")
}

func (b *BpfMapTableView) buildMapTableEditForm() {
	b.form.AddInputField("Key", "", 20, nil, nil).
	AddInputField("Value", "", 20, nil, nil).
	AddButton("Save", func() {
		// Get the new text value that the use input or the old one if they didn't change it
		keyText := b.form.GetFormItemByLabel("Key").(*tview.InputField).GetText()
		valueText := b.form.GetFormItemByLabel("Value").(*tview.InputField).GetText()

		cmd := strings.Split("sudo " + utils.BpftoolPath + " map update id " + strconv.Itoa(b.Map.Id) + " key " + cellTextToHexString(keyText) + " value " + cellTextToHexString(valueText), " ")
		_, _, err := utils.RunCmd(cmd...)
		if err != nil {
			logger.Printf("Error updating map entry: %v\n", err)
		}

		// Create new cells so we can update the table
		newKey := tview.NewTableCell(keyText)
		newValue := tview.NewTableCell(valueText)
		
		// Update the table view
		row, _ := b.table.GetSelection()
		b.table.SetCell(row, 1, newKey)
		b.table.SetCell(row, 2, newValue)

		b.pages.SwitchToPage("table")
	}).
	AddButton("Cancel", func() {
		b.pages.SwitchToPage("table")
	})
}

func (b *BpfMapTableView) buildConfirmModal() {
	b.confirm = tview.NewModal().
		SetText("Are you sure you want to delete this map entry?").
		AddButtons([]string{"Yes", "No"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Yes" {
				row, _ := b.table.GetSelection()
				key := cellTextToHexString(b.table.GetCell(row, 1).Text)
				cmd := strings.Split("sudo " + utils.BpftoolPath + " map delete id " + strconv.Itoa(b.Map.Id) + " key " + key, " ")
				_, _, err := utils.RunCmd(cmd...)
				if err != nil {
					logger.Printf("Error deleting map entry: %v\n", err)
				}
				b.table.RemoveRow(row)
			}
			b.pages.SwitchToPage("table")

		})
}

// Make a new BpfMapTableView. These functions only need to be called once
func NewBpfMapTableView() *BpfMapTableView {
	b := BpfMapTableView{
		form: tview.NewForm(),
		table: tview.NewTable(),
		pages: tview.NewPages(),
	}

	b.buildMapTableView()
	b.buildMapTableEditForm()
	b.buildConfirmModal()

	b.pages.AddPage("table", b.table, true, true)
	b.pages.AddPage("form", b.form, true, false)
	b.pages.AddPage("confirm", b.confirm, true, false)
	
	return &b
}



