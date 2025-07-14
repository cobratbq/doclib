package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/cobratbq/doclib/internal/repo"
	"github.com/cobratbq/goutils/std/builtin"
	"github.com/cobratbq/goutils/std/errors"
	io_ "github.com/cobratbq/goutils/std/io"
	"github.com/cobratbq/goutils/std/log"
)

func queryDocumentName(parent fyne.Window) (string, error) {
	inputName := widget.NewEntry()
	// FIXME perform proper validation of file name
	inputName.Validator = func(s string) error {
		if len(s) > 0 {
			return nil
		}
		return errors.ErrIllegal
	}
	resultChan := make(chan bool, 0)
	dialog.ShowForm("Object name", "Store", "Cancel", []*widget.FormItem{widget.NewFormItem("Name:", inputName)},
		func(proceed bool) {
			log.Traceln("Dialog called with result:", proceed)
			resultChan <- proceed
		}, parent)
	if !<-resultChan {
		return "", errors.Context(errors.ErrFailure, "Dialog cancelled")
	}
	return inputName.Text, nil
}

func constructUI(parent fyne.Window, docrepo *repo.Repo) *fyne.Container {
	lblStatus := widget.NewLabel("")
	lblStatus.TextStyle.Italic = true
	lblStatus.Truncation = fyne.TextTruncateEllipsis
	repoObjs := builtin.Expect(docrepo.List())
	// TODO needs smaller font, more suitable theme, or plain (unthemed) widgets.
	listObjects := widget.NewList(func() int { return len(repoObjs) }, func() fyne.CanvasObject {
		// note: dictate size with wide initial label text at creation
		return widget.NewLabel("XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX")
	}, func(id widget.ListItemID, obj fyne.CanvasObject) {
		obj.(*widget.Label).SetText(repoObjs[id].Props[repo.PROP_NAME])
	})
	// TODO now needs to be sync with repo.Props() list
	interop := struct {
		id   int
		hash binding.String
		name binding.String
	}{id: -1, hash: binding.NewString(), name: binding.NewString()}
	lblHash := widget.NewLabel("hash:")
	lblHash.TextStyle.Italic = true
	lblHashValue := widget.NewLabel("")
	lblHashValue.Bind(interop.hash)
	lblHashValue.Truncation = fyne.TextTruncateEllipsis
	lblName := widget.NewLabel("name:")
	lblName.TextStyle.Italic = true
	inputName := widget.NewEntryWithData(interop.name)
	inputName.Scroll = fyne.ScrollHorizontalOnly
	listObjects.OnSelected = func(id widget.ListItemID) {
		interop.id = id
		interop.hash.Set(repoObjs[id].Props[repo.PROP_HASH])
		interop.name.Set(repoObjs[id].Props[repo.PROP_NAME])
	}
	btnUpdate := widget.NewButton("Update", func() {
		repoObjs[interop.id].Props[repo.PROP_HASH] = builtin.Expect(interop.hash.Get())
		repoObjs[interop.id].Props[repo.PROP_NAME] = builtin.Expect(interop.name.Get())
		if err := docrepo.Save(repoObjs[interop.id]); err != nil {
			lblStatus.SetText("Failed to save updated properties: " + err.Error())
		}
	})
	btnAcquire := widget.NewButton("Acquire", func() {
		openDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				log.Traceln("Failed to open file, possibly dialog cancelled: ", err.Error())
				lblStatus.SetText("Dialog aborted. No file was selected.")
				return
			}
			defer io_.CloseLogged(reader, "Failed to gracefully close file.")
			log.Warnln("File-dialog:", reader.URI(), err)
			// FIXME make CopyFrom copy to temporary location, then wait for file to complete acquisition
			if _, err = docrepo.Acquire(reader, reader.URI().Name()); err == nil {
				log.Traceln("File-dialog successfully completed.")
			} else {
				log.Traceln("Failed to copy document into repository:", err.Error())
				lblStatus.SetText("Failed to import document into repository: " + err.Error())
			}
		}, parent)
		// FIXME fine-tune open-file dialog.
		//openDialog.SetTitleText()
		openDialog.Show()
		listObjects.Refresh()

		// FIXME WIP, continue with multi-step document acquisition ...

		log.Traceln("Document acquired.")
	})
	btnCheck := widget.NewButton("Check", func() {
		// FIXME no error handling
		if err := docrepo.Check(); err == nil {
			lblStatus.SetText("Check finished without errors.")
		} else {
			lblStatus.SetText("Check finished with errors: " + err.Error())
		}
	})
	return container.NewBorder(nil, lblStatus, nil, nil, container.NewBorder(nil, container.NewHBox(btnAcquire, btnCheck), listObjects, nil, container.New(layout.NewFormLayout(), lblHash, lblHashValue, lblName, inputName, layout.NewSpacer(), container.NewBorder(nil, nil, nil, btnUpdate, nil))))
}

func main() {
	// TODO needs an App ID
	app := app.NewWithID("NeedsAnAppID")

	docrepo := repo.OpenRepo("./data")

	mainwnd := app.NewWindow("Doclib")
	mainwnd.SetPadded(false)
	mainwnd.Resize(fyne.NewSize(640, 480))
	mainwnd.SetContent(constructUI(mainwnd, &docrepo))
	//mainwnd.SetOnClosed(func() {})
	mainwnd.ShowAndRun()
}
