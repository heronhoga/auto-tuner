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
    samples := r.widget.RingBuffer.ReadLatest(len(r.lines) + 1)

    if len(samples) < 2 {
        return
    }

    width := size.Width
    height := size.Height

    count := len(samples)
    if count > len(r.lines)+1 {
        count = len(r.lines) + 1
    }

    for i := 0; i < count-1; i++ {
        x1 := float32(i) * width / float32(count-1)
        x2 := float32(i+1) * width / float32(count-1)

        y1 := (1-samples[i]) * height / 2
        y2 := (1-samples[i+1]) * height / 2

        r.lines[i].Position1 = fyne.NewPos(x1, y1)
        r.lines[i].Position2 = fyne.NewPos(x2, y2)
    }
}

func (r *Renderer) MinSize() fyne.Size {
    return fyne.NewSize(400, 200)
}

func (r *Renderer) Refresh() {
    r.Layout(r.widget.Size())

    for _, line := range r.lines {
        line.Refresh()
    }
}