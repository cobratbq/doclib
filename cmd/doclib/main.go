package main

import (
	"flag"
	"strings"

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

type interopType struct {
	id   int
	hash binding.String
	name binding.String
	tags map[string]map[string]binding.Bool
}

func generateTagsContainer(group string, interop *interopType, docrepo *repo.Repo) *fyne.Container {
	lblTag := widget.NewLabel(strings.ToTitle(group) + ":")
	lblTag.TextStyle.Italic = true
	containerTags := container.NewVBox()
	for _, e := range docrepo.Tags(group) {
		chk := widget.NewCheckWithData(e, interop.tags[group][e])
		containerTags.Add(chk)
	}
	return containerTags
}

func constructUI(parent fyne.Window, docrepo *repo.Repo) *fyne.Container {
	repoObjs := builtin.Expect(docrepo.List())
	// TODO needs smaller font, more suitable theme, or plain (unthemed) widgets.
	listObjects := widget.NewList(func() int { return len(repoObjs) }, func() fyne.CanvasObject {
		// note: dictate size with wide initial label text at creation
		return widget.NewLabel("XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX")
	}, func(id widget.ListItemID, obj fyne.CanvasObject) {
		obj.(*widget.Label).SetText(repoObjs[id].Props[repo.PROP_NAME])
	})
	// TODO now needs to be sync with repo.Props() list
	interop := interopType{id: -1, hash: binding.NewString(), name: binding.NewString(), tags: map[string]map[string]binding.Bool{}}
	for _, cat := range docrepo.Categories() {
		interop.tags[cat] = map[string]binding.Bool{}
		for _, tag := range docrepo.Tags(cat) {
			interop.tags[cat][tag] = binding.NewBool()
		}
	}
	lblStatus := widget.NewLabel("")
	lblStatus.TextStyle.Italic = true
	lblStatus.Truncation = fyne.TextTruncateEllipsis
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
		for cat, tags := range interop.tags {
			for k, v := range tags {
				if builtin.Expect(v.Get()) {
					set.Insert(repoObjs[interop.id].Tags[cat], k)
				} else {
					set.Remove(repoObjs[interop.id].Tags[cat], k)
				}
			}
		}
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
			// FIXME quick & dirty solution to refreshing the list after adding a document.
			repoObjs = builtin.Expect(docrepo.List())
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
	tabsTags := container.NewAppTabs()
	categories := map[string]*fyne.Container{}
	for _, cat := range docrepo.Categories() {
		containerCategory := generateTagsContainer(cat, &interop, docrepo)
		categories[cat] = containerCategory
		tabsTags.Items = append(tabsTags.Items, container.NewTabItem(strings.ToTitle(cat), containerCategory))
	}
	tabsTags.Refresh()
	// FIXME support deselecting, zero selections, appropriately clearing values
	listObjects.OnSelected = func(id widget.ListItemID) {
		interop.id = id
		interop.hash.Set(repoObjs[id].Props[repo.PROP_HASH])
		interop.name.Set(repoObjs[id].Props[repo.PROP_NAME])
		for cat, tags := range interop.tags {
			for k, v := range tags {
				_, ok := repoObjs[id].Tags[cat][k]
				v.Set(ok)
			}
		}
		//for _, cat := range docrepo.Categories() {
		//	interop.tags[cat] = map[string]binding.Bool{}
		//	for t := range repoObjs[id].Tags[cat] {
		//		interop.tags[cat][t].Set()
		//		set.Insert(interop.tags[cat], t)
		//	}
		//}
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
				tabsTags,
			),
		),
	)
}

func main() {
	flagRepo := flag.String("repo", "./data", "Location of the repository.")
	flag.Parse()

	// TODO needs an App ID
	app := app.NewWithID("NeedsAnAppID")

	docrepo := repo.OpenRepo(*flagRepo)

	mainwnd := app.NewWindow("Doclib")
	mainwnd.SetPadded(false)
	mainwnd.Resize(fyne.NewSize(640, 480))
	mainwnd.SetContent(constructUI(mainwnd, &docrepo))
	//mainwnd.SetOnClosed(func() {})
	mainwnd.ShowAndRun()
}
