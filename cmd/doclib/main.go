package main

import (
	"flag"
	"slices"
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

func extractRepoObjectList(docrepo *repo.Repo) []repo.RepoObj {
	repoObjs := builtin.Expect(docrepo.List())
	slices.SortFunc(repoObjs, func(a, b repo.RepoObj) int { return strings.Compare(a.Name, b.Name) })
	return repoObjs
}

func generateTagsContainer(group string, interop *interopType, docrepo *repo.Repo) *fyne.Container {
	lblTag := widget.NewLabel(strings.ToTitle(group) + ":")
	lblTag.TextStyle.Italic = true
	containerTags := container.NewVBox()
	for _, tag := range docrepo.Tags(group) {
		chk := widget.NewCheckWithData(tag.Title, interop.tags[group][tag.Key])
		containerTags.Add(chk)
	}
	return containerTags
}

func constructUI(parent fyne.Window, docrepo *repo.Repo) *fyne.Container {
	objects := extractRepoObjectList(docrepo)
	// TODO needs smaller font, more suitable theme, or plain (unthemed) widgets.
	listObjects := widget.NewList(func() int { return len(objects) }, func() fyne.CanvasObject {
		// note: dictate size with wide initial label text at creation
		return widget.NewLabel("XXXXXXXXXXXXXXXXXXXXXXXXXX")
	}, func(id widget.ListItemID, obj fyne.CanvasObject) {
		obj.(*widget.Label).SetText(objects[id].Name)
	})
	// TODO now needs to be sync with repo.Props() list
	interop := interopType{id: -1, hash: binding.NewString(), name: binding.NewString(), tags: map[string]map[string]binding.Bool{}}
	for _, cat := range docrepo.Categories() {
		interop.tags[cat] = map[string]binding.Bool{}
		for _, tag := range docrepo.Tags(cat) {
			interop.tags[cat][tag.Key] = binding.NewBool()
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
		objects[interop.id].Name = builtin.Expect(interop.name.Get())
		for cat, tags := range interop.tags {
			for k, v := range tags {
				if builtin.Expect(v.Get()) {
					set.Insert(objects[interop.id].Tags[cat], k)
				} else {
					set.Remove(objects[interop.id].Tags[cat], k)
				}
			}
		}
		if err := docrepo.Save(objects[interop.id]); err != nil {
			log.Traceln("Failed to save repo-object:", err.Error())
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
			// TODO quick & dirty solution to refreshing the list after adding a document.
			objects = extractRepoObjectList(docrepo)
			listObjects.Refresh()
			log.Traceln("Document import completed.")
		}, parent)
		importDialog.SetConfirmText("Import")
		importDialog.SetTitleText("Import document into ")
		importDialog.Show()
	})
	btnRemove := widget.NewButton("Remove", func() {
		objname := objects[interop.id].Name
		confirmDialog := dialog.NewConfirm("Remove repository object", "Do you want to remove "+objname+" from repository?",
			func(b bool) {
				if !b {
					return
				}
				if err := docrepo.Delete(objects[interop.id].Id); err != nil {
					lblStatus.SetText("Failed to delete object: " + err.Error())
					return
				}
				lblStatus.SetText("Repository object deleted.")
			}, parent)
		confirmDialog.SetConfirmText("Delete")
		confirmDialog.Show()
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
		tabsTags.Items = append(tabsTags.Items, container.NewTabItem(strings.ToTitle(cat), container.NewVScroll(containerCategory)))
	}
	tabsTags.Refresh()
	// FIXME support deselecting, zero selections, appropriately clearing values
	listObjects.OnSelected = func(id widget.ListItemID) {
		interop.id = id
		interop.hash.Set(objects[id].Id)
		interop.name.Set(objects[id].Name)
		for cat, tags := range interop.tags {
			for k, v := range tags {
				_, ok := objects[id].Tags[cat][k]
				v.Set(ok)
			}
		}
	}
	// TODO long-term, it seems the Tags-tabs don't optimally use vertical space yet.
	return container.NewStack(container.NewHSplit(
		container.NewBorder(nil, container.NewHBox(btnAcquire, btnRemove, btnCheck), nil, nil, listObjects),
		container.NewBorder(nil, lblStatus, nil, nil,
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
		)))
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
