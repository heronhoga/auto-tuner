package oscilloscope

import (
	"fyne.io/fyne/v2/widget"
)

type Widget struct {
	widget.BaseWidget
	Model *Model
}

func New(model *Model) *Widget {
	w := &Widget{
		Model: model,
	}

	w.ExtendBaseWidget(w)

	return w
}
