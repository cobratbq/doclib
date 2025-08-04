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

var _ fyne.Layout = (*significantOnTop)(nil)

type significantOnTop struct {
	significant func(obj fyne.CanvasObject) bool
}

func NewSignificantOnTop(significant func(obj fyne.CanvasObject) bool) fyne.Layout {
	return &significantOnTop{
		significant: significant,
	}
}

func (l *significantOnTop) MinSize(objects []fyne.CanvasObject) fyne.Size {
	max := maxMinSize(objects)
	return fyne.NewSize(max.Width, sumHeight(objects))
}

func (l *significantOnTop) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	max := maxMinSize(objects)
	var significant []fyne.CanvasObject
	var plain []fyne.CanvasObject
	for _, o := range objects {
		if !o.Visible() {
			continue
		}
		o.Resize(fyne.NewSize(max.Width, max.Height))
		if l.significant(o) {
			significant = append(significant, o)
		} else {
			plain = append(plain, o)
		}
	}
	i := 0
	for _, o := range significant {
		o.Move(fyne.NewPos(0, float32(i)*max.Height))
		i++
	}
	for _, o := range plain {
		o.Move(fyne.NewPos(0, float32(i)*max.Height))
		i++
	}
}
