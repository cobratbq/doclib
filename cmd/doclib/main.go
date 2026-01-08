// SPDX-License-Identifier: GPL-3.0-only

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
	"github.com/cobratbq/goutils/std/errors"
	io_ "github.com/cobratbq/goutils/std/io"
	"github.com/cobratbq/goutils/std/log"
)

type interopType struct {
	id   binding.Int
	hash binding.String
	name binding.String
	tags map[string]map[string]binding.Bool
}

func (i *interopType) valid() bool {
	return builtin.Expect(i.id.Get()) >= 0 &&
		len(builtin.Expect(i.name.Get())) > 0 &&
		!strings.ContainsAny(builtin.Expect(i.name.Get()), string([]byte{0, '/'}))
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

func backgroundUpdate(docrepo *repo.Repo, btnCheck *widget.Button, updateStatus func(string, widget.Importance)) {
	defer log.Traceln("UI update-button background thread finished.")
	err := docrepo.Check()
	fyne.DoAndWait(func() {
		if err == nil {
			updateStatus("Check finished.", widget.MediumImportance)
			btnCheck.Importance = widget.LowImportance
		} else {
			updateStatus("Check finished with errors: "+err.Error(), widget.MediumImportance)
			btnCheck.Importance = widget.WarningImportance
		}
		btnCheck.Enable()
	})
}

func constructUI(app fyne.App, parent fyne.Window, docrepo *repo.Repo) *fyne.Container {
	objects := repo.ExtractRepoObjectsSorted(docrepo)
	viewmodel := interopType{
		id:   binding.NewInt(),
		hash: binding.NewString(),
		name: binding.NewString(),
		tags: createViewmodelTags(docrepo),
	}
	viewmodel.id.Set(-1)
	// UI components and interaction.
	lblStatus := widget.NewLabel("")
	lblStatus.TextStyle.Italic = true
	lblStatus.Truncation = fyne.TextTruncateEllipsis
	updateStatus := func(text string, importance widget.Importance) {
		lblStatus.Importance = importance
		lblStatus.SetText(text)
	}
	tabsTags := container.NewAppTabs()
	tabsTags.Items = generateTagsTabs(docrepo, &viewmodel)
	tabsTags.OnSelected = func(ti *container.TabItem) {
		ti.Content.(*container.Scroll).Offset = fyne.Position{X: 0, Y: 0}
	}
	tabsTags.Refresh()
	// TODO needs smaller font, more suitable theme, or plain (unthemed) widgets.
	listObjects := widget.NewList(func() int { return len(objects) }, func() fyne.CanvasObject {
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
		app.Clipboard().SetContent(objects[builtin.Expect(viewmodel.id.Get())].Id)
	})
	btnCopyHash.Importance = widget.LowImportance
	lblName := widget.NewLabel("name:")
	lblName.TextStyle.Italic = true
	inputName := widget.NewEntryWithData(viewmodel.name)
	inputName.Scroll = fyne.ScrollHorizontalOnly
	btnOpenRepoLocation := widget.NewButtonWithIcon("", theme.FolderOpenIcon(), func() {
		cmd := exec.Command("xdg-open", docrepo.Location())
		if err := cmd.Start(); err != nil {
			log.Warnln("Failed to open repository location:", err.Error())
			updateStatus("Failed to open repository location: "+err.Error(), widget.WarningImportance)
		}
	})
	btnOpenRepoLocation.Importance = widget.LowImportance
	btnCheck := widget.NewButtonWithIcon("Check", theme.ViewRefreshIcon(), nil)
	btnCheck.OnTapped = func() {
		updateStatus("Checking repository…", widget.MediumImportance)
		btnCheck.Disable()
		go backgroundUpdate(docrepo, btnCheck, updateStatus)
	}
	btnCheck.Importance = widget.LowImportance
	btnOpen := widget.NewButtonWithIcon("", theme.MediaPlayIcon(), func() {
		cmd := exec.Command("/usr/bin/xdg-open", docrepo.ObjectPath(objects[builtin.Expect(viewmodel.id.Get())].Id))
		if err := cmd.Start(); err != nil {
			log.Warnln("Failed to start/open repository object:", err.Error())
			updateStatus("Failed to open repository object: "+err.Error(), widget.WarningImportance)
		}
	})
	btnSave := widget.NewButtonWithIcon("Save", theme.ConfirmIcon(), func() {
		idx := builtin.Expect(viewmodel.id.Get())
		objects[idx].Name = builtin.Expect(viewmodel.name.Get())
		for cat, tags := range viewmodel.tags {
			for k, v := range tags {
				if builtin.Expect(v.Get()) {
					docrepo.Tag(cat, k, &objects[idx])
				} else {
					docrepo.Untag(cat, k, &objects[idx])
				}
			}
		}
		if err := docrepo.Save(objects[idx]); err != nil {
			log.Traceln("Failed to save repo-object:", err.Error())
			updateStatus("Failed to save updated properties: "+err.Error(), widget.WarningImportance)
			return
		}
		listObjects.RefreshItem(idx)
		btnCheck.Importance = widget.HighImportance
		btnCheck.Refresh()
	})
	inputName.Validator = func(s string) error {
		if len(s) > 0 && !strings.ContainsAny(s, string([]byte{0, '/'})) {
			return nil
		} else {
			return errors.ErrIllegal
		}
	}
	btnImport := widget.NewButtonWithIcon("Import", theme.ContentAddIcon(), func() {
		importDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				log.Warnln("Error opening file-dialog: ", err.Error())
				updateStatus("File-dialog failed.", widget.WarningImportance)
				return
			}
			if reader == nil {
				log.Traceln("Document import was cancelled by user.")
				updateStatus("", widget.WarningImportance)
				return
			}
			defer io_.CloseLogged(reader, "Failed to gracefully close file.")
			if newobj, err := docrepo.Acquire(reader, reader.URI().Name()); err == nil {
				log.Traceln("Import-dialog successfully completed.")
				objects = repo.ExtractRepoObjectsSorted(docrepo)
				listObjects.UnselectAll()
				if id := repo.IndexObjectByID(objects, newobj.Id); id >= 0 {
					listObjects.Select(id)
				}
				log.Traceln("Document import completed.")
			} else {
				log.Traceln("Failed to copy document into repository:", err.Error())
				updateStatus("Failed to import document into repository: "+err.Error(), widget.WarningImportance)
			}
		}, parent)
		importDialog.SetTitleText("Import document into ")
		importDialog.SetConfirmText("Import")
		importDialog.Resize(fyne.Size{Width: 800, Height: 600})
		importDialog.Show()
	})
	btnRemove := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		idx := builtin.Expect(viewmodel.id.Get())
		objname := objects[idx].Name
		confirmDialog := dialog.NewConfirm("Remove repository object", "Do you want to remove '"+objname+"'?",
			func(b bool) {
				if !b {
					return
				}
				if err := docrepo.Delete(objects[idx].Id); err != nil {
					log.Warnln("Repository object deletion failed:", err.Error())
					updateStatus("Failed to delete object: "+err.Error(), widget.WarningImportance)
					return
				}
				objects = repo.ExtractRepoObjectsSorted(docrepo)
				listObjects.UnselectAll()
				log.Infoln("Repository object deleted.")
				updateStatus("Repository object deleted.", widget.MediumImportance)
			}, parent)
		confirmDialog.SetConfirmText("Delete")
		confirmDialog.SetDismissText("Cancel")
		confirmDialog.Show()
	})
	btnRemove.Importance = widget.LowImportance
	listObjects.OnSelected = func(id widget.ListItemID) {
		if id < 0 {
			viewmodel.id.Set(-1)
			viewmodel.hash.Set("")
			viewmodel.name.Set("")
			for _, tags := range viewmodel.tags {
				for _, v := range tags {
					v.Set(false)
				}
			}
		} else {
			viewmodel.id.Set(id)
			viewmodel.hash.Set(objects[id].Id)
			viewmodel.name.Set(objects[id].Name)
			for cat, tags := range viewmodel.tags {
				for k, v := range tags {
					v.Set(docrepo.Tagged(cat, k, &objects[id]))
				}
			}
		}
		if tabsTags.SelectedIndex() >= 0 {
			tabsTags.Selected().Content.(*container.Scroll).Offset = fyne.Position{X: 0, Y: 0}
		}
		fyne.Do(tabsTags.Refresh)
	}
	listObjects.OnUnselected = func(id widget.ListItemID) {
		viewmodel.id.Set(-1)
		viewmodel.hash.Set("")
		viewmodel.name.Set("")
	}
	viewmodel.id.AddListener(binding.NewDataListener(func() {
		if id, err := viewmodel.id.Get(); err == nil && id >= 0 {
			btnCopyHash.Enable()
			btnOpen.Enable()
			btnRemove.Enable()
		} else {
			btnCopyHash.Disable()
			btnOpen.Disable()
			btnRemove.Disable()
		}
	}))
	validateOnChanged := binding.NewDataListener(func() {
		if viewmodel.valid() {
			btnSave.Enable()
		} else {
			btnSave.Disable()
		}
	})
	viewmodel.id.AddListener(validateOnChanged)
	viewmodel.hash.AddListener(validateOnChanged)
	viewmodel.name.AddListener(validateOnChanged)
	parent.SetMainMenu(fyne.NewMainMenu(fyne.NewMenu("File", fyne.NewMenuItem("Reload", func() {
		// note: keeping this blocking as most UI content is dependent on this process anyways.
		if err := docrepo.Reload(); err != nil {
			log.Warnln("Failed to reload repository: " + err.Error())
			updateStatus("Failed to reload repository: "+err.Error(), widget.WarningImportance)
			return
		}
		listObjects.UnselectAll()
		objects = repo.ExtractRepoObjectsSorted(docrepo)
		viewmodel.tags = createViewmodelTags(docrepo)
		log.Infoln("Repository reloaded.")
		tabsTags.Items = generateTagsTabs(docrepo, &viewmodel)
		updateStatus("Repository reloaded.", widget.MediumImportance)
		parent.Content().Refresh()
	}))))
	split := container.NewHSplit(
		container.NewBorder(nil, container.NewHBox(btnImport, btnRemove, layout.NewSpacer(), btnOpenRepoLocation, btnCheck), nil, nil,
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

	app := app.New()
	mainwnd := app.NewWindow("Doclib")
	mainwnd.SetPadded(false)
	mainwnd.Resize(fyne.NewSize(800, 600))
	mainwnd.SetContent(constructUI(app, mainwnd, &docrepo))
	mainwnd.ShowAndRun()
}
