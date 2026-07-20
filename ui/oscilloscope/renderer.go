package oscilloscope

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
)

type Renderer struct {
	widget *Widget
	lines []*canvas.Line
	objects []fyne.CanvasObject
}

func (r *Renderer) Objects() []fyne.CanvasObject {
    return r.objects
}

func NewRenderer(w *Widget) *Renderer {
	const resolution = 1024
	lines := make([]*canvas.Line, resolution-1)

	objects := make([]fyne.CanvasObject, len(lines))
	for i := range lines {
		line := canvas.NewLine(theme.PrimaryColor())
		line.StrokeWidth = 1

		lines[i] = line
		objects[i] = line
	}

	return &Renderer{
		widget: w,
		lines: lines,
		objects: objects,
	}
}

func (w *Widget) CreateRenderer() fyne.WidgetRenderer {
	r := NewRenderer(w)
	return r
}

func (r *Renderer) Destroy() {
}

func (r *Renderer) Layout(size fyne.Size) {
}

func (r *Renderer) MinSize() fyne.Size {
    return fyne.NewSize(400, 200)
}

func (r *Renderer) Refresh() {
    for _, line := range r.lines {
        canvas.Refresh(line)
    }
}