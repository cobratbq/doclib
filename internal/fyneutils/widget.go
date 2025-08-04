package fyneutils

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// IsCheckChecked checks if a CanvasObject instance is a checked checkbox. It will assume provided object is
// indeed a `widget.Check`.
func IsCheckChecked(obj fyne.CanvasObject) bool {
	return obj.(*widget.Check).Checked
}

func SetStatusLabel(label *widget.Label, text string, importance widget.Importance) {
	label.Importance = importance
	label.SetText(text)
}

// RefreshTab refreshes the selected tab if one is selected.
// note: check `container.AppTabs.Refresh()` first if it is sufficient.
func RefreshTab(tabs *container.AppTabs) {
	if tabs.SelectedIndex() < 0 {
		return
	}
	tabs.Selected().Content.Refresh()
}
