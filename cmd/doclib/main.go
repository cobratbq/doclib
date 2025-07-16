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
	"github.com/cobratbq/goutils/std/builtin/set"
	io_ "github.com/cobratbq/goutils/std/io"
	"github.com/cobratbq/goutils/std/log"
)

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
		tags map[string]map[string]struct{}
	}{id: -1, hash: binding.NewString(), name: binding.NewString(), tags: map[string]map[string]struct{}{
		repo.TAGGROUP_AUTHORS: make(map[string]struct{}),
	}}
	lblHash := widget.NewLabel("hash:")
	lblHash.TextStyle.Italic = true
	lblHashValue := widget.NewLabel("")
	lblHashValue.Bind(interop.hash)
	lblHashValue.Truncation = fyne.TextTruncateEllipsis
	lblName := widget.NewLabel("name:")
	lblName.TextStyle.Italic = true
	inputName := widget.NewEntryWithData(interop.name)
	inputName.Scroll = fyne.ScrollHorizontalOnly
	btnUpdate := widget.NewButton("Update", func() {
		repoObjs[interop.id].Props[repo.PROP_HASH] = builtin.Expect(interop.hash.Get())
		repoObjs[interop.id].Props[repo.PROP_NAME] = builtin.Expect(interop.name.Get())
		repoObjs[interop.id].Tags = interop.tags
		if err := docrepo.Save(repoObjs[interop.id]); err != nil {
			lblStatus.SetText("Failed to save updated properties: " + err.Error())
		}
	})
	btnAcquire := widget.NewButton("Acquire", func() {
		importDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				log.Warnln("Error opening file-dialog: ", err.Error())
				lblStatus.SetText("File-dialog failed.")
				return
			}
			if reader == nil {
				log.Traceln("Document import was cancelled by user.")
				lblStatus.SetText("")
				return
			}
			defer io_.CloseLogged(reader, "Failed to gracefully close file.")
			if _, err = docrepo.Acquire(reader, reader.URI().Name()); err == nil {
				log.Traceln("File-dialog successfully completed.")
			} else {
				log.Traceln("Failed to copy document into repository:", err.Error())
				lblStatus.SetText("Failed to import document into repository: " + err.Error())
			}
			listObjects.Refresh()
			log.Traceln("Document import completed.")
		}, parent)
		importDialog.SetConfirmText("Import")
		importDialog.SetTitleText("Import document into ")
		importDialog.Show()
	})
	btnCheck := widget.NewButton("Check", func() {
		// FIXME no error handling
		if err := docrepo.Check(); err == nil {
			lblStatus.SetText("Check finished without errors.")
		} else {
			lblStatus.SetText("Check finished with errors: " + err.Error())
		}
	})
	lblAuthors := widget.NewLabel("Authors:")
	lblAuthors.TextStyle.Italic = true
	containerAuthors := container.NewVBox()
	for i, e := range docrepo.Tags(repo.TAGGROUP_AUTHORS) {
		chk := widget.NewCheck(e, func(checked bool) {
			var index = i
			if checked {
				set.Insert(interop.tags[repo.TAGGROUP_AUTHORS], docrepo.Tags(repo.TAGGROUP_AUTHORS)[index])
			} else {
				set.Remove(interop.tags[repo.TAGGROUP_AUTHORS], docrepo.Tags(repo.TAGGROUP_AUTHORS)[index])
			}
		})
		containerAuthors.Add(chk)
	}
	listObjects.OnSelected = func(id widget.ListItemID) {
		interop.id = id
		interop.hash.Set(repoObjs[id].Props[repo.PROP_HASH])
		interop.name.Set(repoObjs[id].Props[repo.PROP_NAME])
		interop.tags = map[string]map[string]struct{}{
			repo.TAGGROUP_AUTHORS: map[string]struct{}{},
		}
		for t := range repoObjs[id].Tags[repo.TAGGROUP_AUTHORS] {
			set.Insert(interop.tags[repo.TAGGROUP_AUTHORS], t)
		}
		for _, e := range containerAuthors.Objects {
			chk := e.(*widget.Check)
			_, ok := interop.tags[repo.TAGGROUP_AUTHORS][chk.Text]
			chk.SetChecked(ok)
		}
	}
	return container.NewBorder(nil, lblStatus, nil, nil,
		container.NewBorder(
			nil, container.NewHBox(btnAcquire, btnCheck), listObjects, nil,
			container.NewBorder(
				container.New(layout.NewFormLayout(),
					lblHash, lblHashValue,
					lblName, inputName,
					layout.NewSpacer(), container.NewBorder(nil, nil, nil, btnUpdate, nil),
				),
				nil,
				nil,
				nil,
				container.NewBorder(lblAuthors, nil, nil, nil, containerAuthors),
			),
		),
	)
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
