// This page handles the the view for looking at the map entries of a map. It
// allows the user to select a map and then view the entries in that map. The
// user can also edit the map entries from this view.
package ui

import (
	"ebpfmon/utils"
	"fmt"
	"strconv"
	"strings"
	"encoding/binary"

	log "github.com/sirupsen/logrus"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	Hex = 0
	Decimal = 1
	Char = 2
	Raw = 3
)

const  (
	DataWidth8 = 1
	DataWidth16 = 2
	DataWidth32 = 4
	DataWidth64 = 8
)
const (
	Little = 0
	Big = 1
)

var curFormat = Hex
var curWidth = DataWidth8
var curEndianness = Little

type BpfMapTableView struct {
	pages *tview.Pages
	form *tview.Form
	filter *tview.Form
	table *tview.Table
	confirm *tview.Modal
	Map utils.BpfMap
	MapEntries []utils.BpfMapEntry
}

func asDecimal(width int, endian int, data []byte) string {
	var result string = ""
	switch width {
		case DataWidth8:
			for _, b := range data {
				result += strconv.Itoa(int(b)) + " "
			}
			break
		case DataWidth16:
			if len(data) % DataWidth16 != 0 {
				return ""
			}

			if endian == Little {
				for i := 0; i < len(data); i += 2 {
					result += strconv.Itoa(int(binary.LittleEndian.Uint16(data[i:i+2]))) + " "
				}
			} else {
				for i := 0; i < len(data); i += 2 {
					result += strconv.Itoa(int(binary.BigEndian.Uint16(data[i:i+2]))) + " "
				}
			}
			break
		case DataWidth32:
			if len(data) % DataWidth32 != 0 {
				return ""
			}
			if endian == Little {
				for i := 0; i < len(data); i += 4 {
					result += strconv.Itoa(int(binary.LittleEndian.Uint32(data[i:i+4]))) + " "
				}
			} else {
				for i := 0; i < len(data); i += 4 {
					result += strconv.Itoa(int(binary.BigEndian.Uint32(data[i:i+4]))) + " "
				}
			}
			break
		case DataWidth64:
			if len(data) % DataWidth64 != 0 {
				return ""
			}

			if endian == Little {
				for i := 0; i < len(data); i += 8 {
					result += strconv.Itoa(int(binary.LittleEndian.Uint64(data[i:i+8]))) + " "
				}
			} else {
				for i := 0; i < len(data); i += 8 {
					result += strconv.Itoa(int(binary.BigEndian.Uint64(data[i:i+8]))) + " "
				}
			}
			break
	}
	result = strings.Trim(result, " ")
	return result
}

// Similar to the asDecimal function except it displays the data as hex
func asHex(width int, endian int, data []byte) string {
	var result string = ""
	switch width {
		case DataWidth8:
			for _, b := range data {
				result += fmt.Sprintf("%#02x", b) + " "
			}
			break
		case DataWidth16:
			if len(data) % DataWidth16 != 0 {
				return ""
			}

			if endian == Little {
				for i := 0; i < len(data); i += 2 {
					result += fmt.Sprintf("%#04x", binary.LittleEndian.Uint16(data[i:i+2])) + " "
				}
			} else {
				for i := 0; i < len(data); i += 2 {
					result += fmt.Sprintf("%#04x", binary.BigEndian.Uint16(data[i:i+2])) + " "
				}
			}
			break
		case DataWidth32:
			if len(data) % DataWidth32 != 0 {
				return ""
			}
			if endian == Little {
				for i := 0; i < len(data); i += 4 {
					result += fmt.Sprintf("%#08x", binary.LittleEndian.Uint32(data[i:i+4])) + " "
				}
			} else {
				for i := 0; i < len(data); i += 4 {
					result += fmt.Sprintf("%#08x", binary.BigEndian.Uint32(data[i:i+4])) + " "
				}
			}
			break
		case DataWidth64:
			if len(data) % DataWidth64 != 0 {
				return ""
			}			

			if endian == Little {
				for i := 0; i < len(data); i += 8 {
					result += fmt.Sprintf("%#016x", binary.LittleEndian.Uint64(data[i:i+8])) + " "
				}
			} else {
				for i := 0; i < len(data); i += 8 {
					result += fmt.Sprintf("%#016x", binary.BigEndian.Uint64(data[i:i+8])) + " "
				}
			}
			break
	}
	result = strings.Trim(result, " ")

	return result
}

func asChar(data []byte) string {
	var result string = ""
	for _, b := range data {
		if b >= 32 && b <= 126 {
			result += fmt.Sprintf("%c", b)
		} else {
			result += "."
		}
	}
	return result
}

// Doesn't change the default formatting of the data
func asRaw(data []byte) string {
	return fmt.Sprintf("%v", data)
}

// Adds some null bytes to the beggining of a slice
func padBytesBeginning(data []byte, width int) []byte {
	if len(data) % width == 0 {
		return data
	}

	var bytesNeeded int = width - (len(data) % width)

	var result []byte
	for i := 0; i < bytesNeeded; i++ {
		result = append(result, 0)
	}
	result = append(result, data...)
	return result
}

// Add some null bytes to the end of a slice
func padBytesEnd(data []byte, width int) []byte {
	if len(data) % width == 0 {
		return data
	}

	var bytesNeeded int = width - (len(data) % width)

	var result []byte
	for i := 0; i < bytesNeeded; i++ {
		result = append(result, 0)
	}
	result = append(data, result...)
	return result
}

// Adds padding to the bytes based on endianness
func padBytes(data []byte, width int) []byte {
	if len(data) % width == 0 {
		return data
	}

	return padBytesEnd(data, width)
}

// Apply a format based on the specified format, width, endianness
func applyFormat(format int, width int, endianness int, data []byte) string {
	if len(data) == 0 {
		return ""
	}
	
	data = padBytes(data, width)

	switch format {
	case Hex:
		return asHex(width, endianness, data)
	case Decimal:
		return asDecimal(width, endianness, data)
	case Char:
		return asChar(data)
	default:
		return asRaw(data)
	}
}

// Update the table view with the new map entries
func (b *BpfMapTableView) updateTable() {
	b.table.Clear()
	b.table.SetCell(0, 0, tview.NewTableCell("Index").SetSelectable(false))
	b.table.SetCell(0, 1, tview.NewTableCell("Key").SetSelectable(false))
	b.table.SetCell(0, 2, tview.NewTableCell("Value").SetSelectable(false))
	for i, entry := range b.MapEntries {
		b.table.SetCell(i+1, 0, tview.NewTableCell(strconv.Itoa(i)))
		b.table.SetCell(i+1, 1, tview.NewTableCell(applyFormat(curFormat, curWidth, curEndianness, entry.Key)))
		b.table.SetCell(i+1, 2, tview.NewTableCell(applyFormat(curFormat, curWidth, curEndianness, entry.Value)))
	}
}

// Update Map
func (b *BpfMapTableView) UpdateMap(m utils.BpfMap) {
	var err error
	b.Map = m
	b.MapEntries, err = utils.GetBpfMapEntries(b.Map.Id)
	if err != nil {
		log.Printf("Error getting map entries: %v\n", err)
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
		if len(b.MapEntries) <= 0 {
			return
		}

		key := b.MapEntries[row-1].Key
		value := b.MapEntries[row-1].Value
		b.form.GetFormItemByLabel("Key").(*tview.InputField).SetText(fmt.Sprintf("%v", key))
		b.form.GetFormItemByLabel("Value").(*tview.InputField).SetText(fmt.Sprintf("%v", value))
		b.form.SetFocus(0)
		b.pages.SwitchToPage("form")
	})
}

func cellTextToHexString(cellValue string) string {
	return strings.Trim(cellValue, "[]")
}

func cellTextToByteSlice(cellValue string) []byte {
	trimmed := strings.Trim(cellValue, "[]")
	split := strings.Split(trimmed, " ")
	var result []byte
	for _, s := range split {
		b, err := strconv.ParseUint(s, 0, 8)
		if err != nil {
			log.Printf("Error converting cell text to byte slice: %v\n", err)
			return []byte{}
		}
		result = append(result, byte(b))
	}
	return result
}

func (b *BpfMapTableView) buildMapTableEditForm() {
	b.form.AddInputField("Key", "", 0, nil, nil).
	AddInputField("Value", "", 0, nil, nil).
	AddButton("Save", func() {
		if len(b.MapEntries) <= 0 {
			return
		}

		// Get the new text value that the use input or the old one if they didn't change it
		keyText := b.form.GetFormItemByLabel("Key").(*tview.InputField).GetText()
		valueText := b.form.GetFormItemByLabel("Value").(*tview.InputField).GetText()

		cmd := strings.Split("sudo " + utils.BpftoolPath + " map update id " + strconv.Itoa(b.Map.Id) + " key " + cellTextToHexString(keyText) + " value " + cellTextToHexString(valueText), " ")
		_, _, err := utils.RunCmd(cmd...)
		if err != nil {
			if b.Map.Frozen == 1 {
				log.Errorf("Failed to update map entry becuse the map is fozen: %v\n", err)
			} else {
				log.Errorf("Failed to update map entry: %v\nAttempted cmd: %s", err, cmd)
			}
		}

		// Update the map entries
		row, _ := b.table.GetSelection()
		b.MapEntries[row-1].Key = []byte(cellTextToByteSlice(keyText))
		b.MapEntries[row-1].Value = []byte(cellTextToByteSlice(valueText))
		b.UpdateMap(b.Map)
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
				key := strings.Trim(fmt.Sprintf("%v", b.MapEntries[row-1].Key), "[]")
				cmd := strings.Split("sudo " + utils.BpftoolPath + " map delete id " + strconv.Itoa(b.Map.Id) + " key " + key, " ")
				_, _, err := utils.RunCmd(cmd...)
				if err != nil {
					log.Printf("Error deleting map entry: %v\n", err)
				}

				b.UpdateMap(b.Map)
			}
			b.pages.SwitchToPage("table")
		})
}

func (b *BpfMapTableView) buildFilterForm() {
	b.filter = tview.NewForm().
		AddDropDown("Data Format", []string{"Hex", "Decimal", "Char", "Raw"}, 0, func(option string, optionIndex int) {
			switch optionIndex {
			case 0:
				curFormat = Hex
				break
			case 1:
				curFormat = Decimal
				break
			case 2:
				curFormat = Char
				break
			default:
				curFormat = Raw
			}
			b.updateTable()
		}).AddDropDown("Endianness", []string{"Little", "Big"}, 0, func(option string, optionIndex int) {
			switch optionIndex {
			case 0:
				curEndianness = Little
				break
			default:
				curEndianness = Big
			}
			b.updateTable()
		}).AddDropDown("Data Width", []string{"8", "16", "32", "64"}, 0, func(option string, optionIndex int) {
			switch optionIndex {
			case 0:
				curWidth = DataWidth8
				break
			case 1:
				curWidth = DataWidth16
				break
			case 2:
				curWidth = DataWidth32
				break
			default:
				curWidth = DataWidth64
			}
			b.updateTable()
		})
	
}

// Make a new BpfMapTableView. These functions only need to be called once
func NewBpfMapTableView(tui *Tui) *BpfMapTableView {
	b := BpfMapTableView{
		form: tview.NewForm(),
		table: tview.NewTable(),
		pages: tview.NewPages(),
	}

	b.buildMapTableView()
	b.buildMapTableEditForm()
	b.buildConfirmModal()
	b.buildFilterForm()

	flex := tview.NewFlex().
		AddItem(b.filter, 0, 1, false).
		AddItem(b.table, 0, 3, true)

	flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTAB {
			if b.table.HasFocus() {
				tui.App.SetFocus(b.filter)
				return nil
			} 
		} else if event.Key() == tcell.KeyEsc {
			if b.filter.HasFocus() {
				tui.App.SetFocus(b.table)
				return nil

			}
		}

		return event
	})

	b.pages.AddPage("table", flex, true, true)
	b.pages.AddPage("form", b.form, true, false)
	b.pages.AddPage("confirm", b.confirm, true, false)
	
	return &b
}



