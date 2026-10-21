package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/heronhoga/auto-tuner/utils/effects"
)

func NewEffects(processor *effects.Processor) fyne.CanvasObject {
	settings := processor.Settings()
	newAmountSlider := func(name string, initial float64, changed func(float64)) fyne.CanvasObject {
		label := widget.NewLabel(fmt.Sprintf("%s: %.0f%%", name, initial*100))
		slider := widget.NewSlider(0, 100)
		slider.Step = 1
		slider.SetValue(initial * 100)
		slider.OnChanged = func(value float64) {
			changed(value / 100)
			label.SetText(fmt.Sprintf("%s: %.0f%%", name, value))
		}
		return container.NewVBox(label, slider)
	}

	return container.NewVBox(
		widget.NewSeparator(),
		container.NewGridWithColumns(2,
			newAmountSlider("Chorus", settings.Chorus, func(value float64) {
				settings.Chorus = value
				processor.SetSettings(settings)
			}),
			newAmountSlider("Reverb", settings.Reverb, func(value float64) {
				settings.Reverb = value
				processor.SetSettings(settings)
			}),
		),
		widget.NewLabel("0% = off · 100% = maximum effect · Chorus → Reverb"),
	)
}
