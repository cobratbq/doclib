package main

import (
	"flag"
	"os/exec"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/cobratbq/doclib/internal/repo"
	"github.com/cobratbq/goutils/assert"
	"github.com/cobratbq/goutils/std/builtin"
	"github.com/cobratbq/goutils/std/builtin/set"
	"github.com/cobratbq/goutils/std/errors"
	io_ "github.com/cobratbq/goutils/std/io"
	"github.com/cobratbq/goutils/std/log"
)

type interopType struct {
	id   int
	hash binding.String
	name binding.String
	tags map[string]map[string]binding.Bool
}

// FIXME extend name validation to include illegal file-system symbols or do proper filtering before applying to file-system objects.
func (i *interopType) valid() bool {
	return i.id >= 0 && len(builtin.Expect(i.name.Get())) > 0
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

// TODO consider adding button to reload repository information, and rebuild tags/checkboxes lists with updated dirs/sub-dirs/content.
func constructUI(app fyne.App, parent fyne.Window, docrepo *repo.Repo) *fyne.Container {
	objects := repo.ExtractRepoObjectsSorted(docrepo)
	// TODO needs smaller font, more suitable theme, or plain (unthemed) widgets.
	listObjects := widget.NewList(func() int { return len(objects) }, func() fyne.CanvasObject {
		// note: dictate size with wide initial label text at creation
		return widget.NewLabel("XXXXXXXXXXXXXXXXXXXXXXXXXX")
	}, func(id widget.ListItemID, obj fyne.CanvasObject) {
		obj.(*widget.Label).SetText(objects[id].Name)
	})
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
	btnCopyHash := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
		app.Clipboard().SetContent(objects[interop.id].Id)
	})
	lblName := widget.NewLabel("name:")
	lblName.TextStyle.Italic = true
	inputName := widget.NewEntryWithData(interop.name)
	inputName.Scroll = fyne.ScrollHorizontalOnly
	btnOpen := widget.NewButtonWithIcon("", theme.MediaPlayIcon(), func() {
		cmd := exec.Command("xdg-open", docrepo.ObjectPath(objects[interop.id].Id))
		if err := cmd.Start(); err != nil {
			log.Warnln("Failed to start/open repository object:", err.Error())
			return
		}
	})
	btnUpdate := widget.NewButtonWithIcon("Update", theme.ConfirmIcon(), func() {
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
		listObjects.RefreshItem(interop.id)
	})
	inputName.Validator = func(s string) error {
		if interop.valid() {
			btnOpen.Enable()
			btnUpdate.Enable()
			return nil
		} else {
			btnOpen.Disable()
			btnUpdate.Disable()
			return errors.ErrIllegal
		}
	}
	btnImport := widget.NewButtonWithIcon("Import", theme.ContentAddIcon(), func() {
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
			if newobj, err := docrepo.Acquire(reader, reader.URI().Name()); err == nil {
				log.Traceln("Import-dialog successfully completed.")
				objects = repo.ExtractRepoObjectsSorted(docrepo)
				listObjects.UnselectAll()
				listObjects.Refresh()
				if id := repo.IndexObjectByID(objects, newobj.Id); id >= 0 {
					listObjects.Select(id)
				}
				log.Traceln("Document import completed.")
			} else {
				log.Traceln("Failed to copy document into repository:", err.Error())
				lblStatus.SetText("Failed to import document into repository: " + err.Error())
			}
		}, parent)
		importDialog.SetConfirmText("Import")
		importDialog.SetTitleText("Import document into ")
		importDialog.Resize(fyne.Size{Width: 800, Height: 600})
		importDialog.Show()
	})
	btnRemove := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		if interop.id < 0 {
			return
		}
		objname := objects[interop.id].Name
		confirmDialog := dialog.NewConfirm("Remove repository object", "Do you want to remove '"+objname+"'?",
			func(b bool) {
				if !b {
					return
				}
				if err := docrepo.Delete(objects[interop.id].Id); err != nil {
					log.Traceln("Repository object deletion failed:", err.Error())
					lblStatus.SetText("Failed to delete object: " + err.Error())
					return
				}
				objects = repo.ExtractRepoObjectsSorted(docrepo)
				listObjects.UnselectAll()
				listObjects.Refresh()
				log.Traceln("Repository object deleted.")
				lblStatus.SetText("Repository object deleted.")
			}, parent)
		confirmDialog.SetConfirmText("Delete")
		confirmDialog.SetDismissText("Cancel")
		confirmDialog.Show()
	})
	btnCheck := widget.NewButtonWithIcon("Check", theme.ViewRefreshIcon(), func() {
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
	listObjects.OnUnselected = func(id widget.ListItemID) {
		interop.id = -1
		interop.hash.Set("")
		interop.name.Set("")
		for _, tags := range interop.tags {
			for _, v := range tags {
				v.Set(false)
			}
		}
	}
	// TODO long-term, it seems the Tags-tabs don't optimally use vertical space yet.
	split := container.NewHSplit(
		container.NewBorder(nil, container.NewHBox(btnImport, btnRemove, layout.NewSpacer(), btnCheck), nil, nil,
			listObjects),
		container.NewBorder(nil, lblStatus, nil, nil,
			container.NewBorder(
				container.New(layout.NewFormLayout(),
					lblHash, container.NewBorder(nil, nil, nil, btnCopyHash, lblHashValue),
					lblName, inputName,
					layout.NewSpacer(), container.NewBorder(nil, nil, nil, container.NewHBox(btnOpen, btnUpdate), nil),
				), nil, nil, nil,
				tabsTags,
			),
		))
	split.SetOffset(0.3)
	return container.NewStack(split)
}

func main() {
	flagRepo := flag.String("repo", "./data", "Location of the repository.")
	flag.Parse()

	// TODO needs an App ID
	app := app.NewWithID("NeedsAnAppID")

	docrepo, err := repo.OpenRepo(*flagRepo)
	assert.Success(err, "Failed to open repository at: "+*flagRepo)

	mainwnd := app.NewWindow("Doclib")
	mainwnd.SetPadded(false)
	mainwnd.Resize(fyne.NewSize(800, 600))
	mainwnd.SetContent(constructUI(app, mainwnd, &docrepo))
	//mainwnd.SetOnClosed(func() {})
	mainwnd.ShowAndRun()
}
