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
	"github.com/cobratbq/doclib/internal/fyneutils"
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

func (i *interopType) valid() bool {
	// FIXME extend name validation to include illegal file-system symbols or do proper filtering before applying to file-system objects.
	return i.id >= 0 && len(builtin.Expect(i.name.Get())) > 0 && !strings.ContainsAny(builtin.Expect(i.name.Get()), string([]byte{0, '/'}))
}

func createViewmodelTags(docrepo *repo.Repo) map[string]map[string]binding.Bool {
	tags := map[string]map[string]binding.Bool{}
	for _, cat := range docrepo.Categories() {
		tags[cat] = map[string]binding.Bool{}
		for _, tag := range docrepo.Tags(cat) {
			tags[cat][tag.Key] = binding.NewBool()
		}
	}
	return tags
}

func generateTagsContainer(group string, interop *interopType, docrepo *repo.Repo) *fyne.Container {
	lblTag := widget.NewLabel(strings.ToTitle(group) + ":")
	lblTag.TextStyle.Italic = true
	container := container.New(fyneutils.NewSignificantOnTop(fyneutils.IsCheckChecked))
	for _, tag := range docrepo.Tags(group) {
		chk := widget.NewCheckWithData(tag.Title, interop.tags[group][tag.Key])
		container.Objects = append(container.Objects, chk)
	}
	return container
}

func generateTagsTabs(docrepo *repo.Repo, interop *interopType) []*container.TabItem {
	categories := map[string]*fyne.Container{}
	var items []*container.TabItem
	for _, cat := range docrepo.Categories() {
		containerCategory := generateTagsContainer(cat, interop, docrepo)
		categories[cat] = containerCategory
		items = append(items, container.NewTabItem(strings.ToTitle(cat), container.NewVScroll(containerCategory)))
	}
	return items
}

// TODO consider setting both importance and text for status-label upon changing status text (success, warnings).
// TODO consider adding button to reload repository information, and rebuild tags/checkboxes lists with updated dirs/sub-dirs/content.
func constructUI(app fyne.App, parent fyne.Window, docrepo *repo.Repo) *fyne.Container {
	objects := repo.ExtractRepoObjectsSorted(docrepo)
	viewmodel := interopType{id: -1, hash: binding.NewString(), name: binding.NewString(), tags: map[string]map[string]binding.Bool{}}
	viewmodel.tags = createViewmodelTags(docrepo)
	// UI components and interaction.
	lblStatus := widget.NewLabel("")
	lblStatus.TextStyle.Italic = true
	lblStatus.Truncation = fyne.TextTruncateEllipsis
	tabsTags := container.NewAppTabs()
	tabsTags.Items = generateTagsTabs(docrepo, &viewmodel)
	tabsTags.Refresh()
	// TODO needs smaller font, more suitable theme, or plain (unthemed) widgets.
	listObjects := widget.NewList(func() int { return len(objects) }, func() fyne.CanvasObject {
		// note: dictate size with wide initial label text at creation
		return widget.NewLabel("")
	}, func(id widget.ListItemID, obj fyne.CanvasObject) {
		obj.(*widget.Label).SetText(objects[id].Name)
	})
	lblHash := widget.NewLabel("hash:")
	lblHash.TextStyle.Italic = true
	lblHashValue := widget.NewLabel("")
	lblHashValue.Bind(viewmodel.hash)
	lblHashValue.Truncation = fyne.TextTruncateEllipsis
	btnCopyHash := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
		app.Clipboard().SetContent(objects[viewmodel.id].Id)
	})
	btnCopyHash.Importance = widget.LowImportance
	lblName := widget.NewLabel("name:")
	lblName.TextStyle.Italic = true
	inputName := widget.NewEntryWithData(viewmodel.name)
	inputName.Scroll = fyne.ScrollHorizontalOnly
	btnUpdate := widget.NewButtonWithIcon("Update", theme.ViewRefreshIcon(), nil)
	btnUpdate.OnTapped = func() {
		if err := docrepo.Check(); err == nil {
			lblStatus.SetText("Check finished.")
			btnUpdate.Importance = widget.LowImportance
		} else {
			lblStatus.SetText("Check finished with errors: " + err.Error())
			btnUpdate.Importance = widget.WarningImportance
		}
		btnUpdate.Refresh()
	}
	btnUpdate.Importance = widget.LowImportance
	btnOpen := widget.NewButtonWithIcon("", theme.MediaPlayIcon(), func() {
		cmd := exec.Command("xdg-open", docrepo.ObjectPath(objects[viewmodel.id].Id))
		if err := cmd.Start(); err != nil {
			log.Warnln("Failed to start/open repository object:", err.Error())
			return
		}
	})
	btnSave := widget.NewButtonWithIcon("Save", theme.ConfirmIcon(), func() {
		objects[viewmodel.id].Name = builtin.Expect(viewmodel.name.Get())
		for cat, tags := range viewmodel.tags {
			for k, v := range tags {
				if builtin.Expect(v.Get()) {
					set.Insert(objects[viewmodel.id].Tags[cat], k)
				} else {
					set.Remove(objects[viewmodel.id].Tags[cat], k)
				}
			}
		}
		if err := docrepo.Save(objects[viewmodel.id]); err != nil {
			log.Traceln("Failed to save repo-object:", err.Error())
			lblStatus.SetText("Failed to save updated properties: " + err.Error())
			return
		}
		listObjects.RefreshItem(viewmodel.id)
		btnUpdate.Importance = widget.HighImportance
		btnUpdate.Refresh()
	})
	inputName.Validator = func(s string) error {
		if viewmodel.valid() {
			btnOpen.Enable()
			btnSave.Enable()
			return nil
		} else {
			btnOpen.Disable()
			btnSave.Disable()
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
		if viewmodel.id < 0 {
			return
		}
		objname := objects[viewmodel.id].Name
		confirmDialog := dialog.NewConfirm("Remove repository object", "Do you want to remove '"+objname+"'?",
			func(b bool) {
				if !b {
					return
				}
				if err := docrepo.Delete(objects[viewmodel.id].Id); err != nil {
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
	btnRemove.Importance = widget.LowImportance
	listObjects.OnSelected = func(id widget.ListItemID) {
		if id < 0 {
			viewmodel.id = -1
			viewmodel.hash.Set("")
			viewmodel.name.Set("")
			for _, tags := range viewmodel.tags {
				for _, v := range tags {
					v.Set(false)
				}
			}
		} else {
			viewmodel.id = id
			viewmodel.hash.Set(objects[id].Id)
			viewmodel.name.Set(objects[id].Name)
			for cat, tags := range viewmodel.tags {
				for k, v := range tags {
					_, ok := objects[id].Tags[cat][k]
					v.Set(ok)
				}
			}
		}
		fyne.Do(tabsTags.Refresh)
	}
	parent.SetMainMenu(fyne.NewMainMenu(fyne.NewMenu("File", fyne.NewMenuItem("Reload", func() {
		if err := docrepo.Refresh(); err != nil {
			log.WarnOnError(err, "Failed to reload repository")
			lblStatus.Importance = widget.WarningImportance
			lblStatus.SetText("Failed to reload repository: " + err.Error())
			return
		}
		listObjects.UnselectAll()
		objects = repo.ExtractRepoObjectsSorted(docrepo)
		viewmodel.tags = createViewmodelTags(docrepo)
		log.Infoln("Repository reloaded.")
		tabsTags.Items = generateTagsTabs(docrepo, &viewmodel)
		lblStatus.Importance = widget.MediumImportance
		lblStatus.SetText("Repository reloaded.")
		parent.Content().Refresh()
	}))))
	// TODO long-term, it seems the Tags-tabs don't optimally use vertical space yet.
	split := container.NewHSplit(
		container.NewBorder(nil, container.NewHBox(btnImport, btnRemove, layout.NewSpacer(), btnUpdate), nil, nil,
			listObjects),
		container.NewBorder(
			container.New(layout.NewFormLayout(),
				lblHash, container.NewBorder(nil, nil, nil, btnCopyHash, lblHashValue),
				lblName, inputName,
				layout.NewSpacer(), container.NewBorder(nil, nil, nil, container.NewHBox(btnOpen, btnSave), nil),
			), lblStatus, nil, nil,
			tabsTags,
		),
	)
	split.SetOffset(0.3)
	return container.NewStack(split)
}

func main() {
	flagRepo := flag.String("repo", "./data", "Location of the repository.")
	flag.Parse()

	docrepo, err := repo.OpenRepository(*flagRepo)
	assert.Success(err, "Failed to open repository at: "+*flagRepo)

	// TODO needs a proper App-ID
	app := app.NewWithID("NeedsAnAppID")
	mainwnd := app.NewWindow("Doclib")
	mainwnd.SetPadded(false)
	mainwnd.Resize(fyne.NewSize(800, 600))
	mainwnd.SetContent(constructUI(app, mainwnd, &docrepo))
	mainwnd.ShowAndRun()
}
