package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/cobratbq/doclib/internal/repo"
	"github.com/cobratbq/goutils/std/errors"
)

func constructUI(parent fyne.Window, repo *repo.Repo) *fyne.Container {
	lblStatus := widget.NewLabel("")
	lblStatus.TextStyle.Italic = true
	listObjects := widget.NewList(repo.Len, func() fyne.CanvasObject {
		return widget.NewLabel("new item")
	}, func(id widget.ListItemID, obj fyne.CanvasObject) {
		item := obj.(*widget.Label)
		item.SetText("Hello world")
	})
	listProps := widget.NewList(repo.Len, func() fyne.CanvasObject {
		return container.NewHBox(widget.NewLabel("new property"), widget.NewEntry())
	}, func(id widget.ListItemID, obj fyne.CanvasObject) {
	})
	btnAcquire := widget.NewButton("Acquire", func() {
		inputName := widget.NewEntry()
		inputName.Validator = func(s string) error {
			if len(s) > 0 {
				return nil
			}
			return errors.ErrIllegal
		}
		dialog.ShowForm("Object name", "Store", "Cancel", []*widget.FormItem{widget.NewFormItem("Name:", inputName)}, func(b bool) {
			// FIXME hardcoded path
			if err := repo.Acquire("/home/dev-otr/mydocument", inputName.Text); err == nil {
				lblStatus.SetText("")
			} else {
				lblStatus.SetText("ACQUIRE: " + err.Error())
			}
		}, parent)
	})
	btnCheck := widget.NewButton("Check", func() {
		// FIXME no error handling
		if err := repo.Check(); err == nil {
			lblStatus.SetText("")
		} else {
			lblStatus.SetText("CHECK: " + err.Error())
		}
	})
	return container.NewBorder(nil, lblStatus, listObjects, nil, container.NewVBox(listProps, widget.NewLabel("Testing..."), btnAcquire, btnCheck))
}

func main() {
	app := app.New()

	docrepo := repo.OpenRepo("./data")

	mainwnd := app.NewWindow("Doclib")
	mainwnd.SetPadded(false)
	mainwnd.Resize(fyne.NewSize(640, 480))
	mainwnd.SetContent(constructUI(mainwnd, &docrepo))
	//mainwnd.SetOnClosed(func() {})
	mainwnd.ShowAndRun()
}
