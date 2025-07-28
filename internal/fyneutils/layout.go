package fyneutils

import (
	"fyne.io/fyne/v2"
	"github.com/cobratbq/goutils/std/builtin/slices"
	"github.com/cobratbq/goutils/std/log"
)

func maxMinSize(objects []fyne.CanvasObject) fyne.Size {
	return slices.Fold(objects, fyne.NewSize(0, 0), func(v fyne.Size, obj fyne.CanvasObject) fyne.Size {
		return v.Max(obj.MinSize())
	})
}

func sumHeight(objects []fyne.CanvasObject) float32 {
	return slices.Fold(objects, 0, func(v float32, e fyne.CanvasObject) float32 {
		return v + e.MinSize().Height
	})
}

var _ fyne.Layout = (*hOverflowLayout)(nil)

type hOverflowLayout struct {
	maxCols uint
	numCols int
}

func NewHOverflowLayout(maxCols uint) fyne.Layout {
	return &hOverflowLayout{maxCols: maxCols}
}

func (l *hOverflowLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	max := maxMinSize(objects)
	// FIXME assuming numCols will force a certain minimum width that is larger than single column, but is useful to establish propert width/height. (How to make it work in ideal way for all cases?)
	//maxwidth := float32(l.numCols) * max.Width
	//maxheight := sumHeight(objects)/float32(l.numCols) + max.Height
	return fyne.NewSize(max.Width, sumHeight(objects))
}

func (l *hOverflowLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	max := maxMinSize(objects)
	l.numCols = int(l.maxCols)
	for float32(l.numCols)*max.Width > size.Width {
		l.numCols--
	}
	if l.numCols == 0 {
		log.Traceln("numCols:", l.numCols)
		return
	}
	i, perCol := 0, len(objects)/l.numCols+1
	for _, o := range objects {
		if !o.Visible() {
			continue
		}
		col, row := float32(i/perCol), float32(i%perCol)
		o.Resize(fyne.NewSize(max.Width, max.Height))
		o.Move(fyne.NewPos(col*max.Width, row*max.Height))
		i++
	}
	log.Traceln("Canvas:", size)
}
