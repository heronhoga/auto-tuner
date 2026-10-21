package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/heronhoga/auto-tuner/utils/tune"
)

func NewPitchCorrection(corrector *tune.AutoCorrector) fyne.CanvasObject {
	settings := corrector.Settings()
	enabled := widget.NewCheck("Pitch correction", func(value bool) {
		settings.Enabled = value
		corrector.SetSettings(settings)
	})
	enabled.SetChecked(settings.Enabled)

	keys := []string{"C", "C# / Db", "D", "D# / Eb", "E", "F", "F# / Gb", "G", "G# / Ab", "A", "A# / Bb", "B"}
	key := widget.NewSelect(keys, nil)
	key.SetSelectedIndex(settings.Key)
	key.OnChanged = func(_ string) {
		settings.Key = key.SelectedIndex()
		corrector.SetSettings(settings)
	}

	scale := widget.NewSelect([]string{string(tune.Major), string(tune.Minor)}, func(value string) {
		settings.Scale = tune.Scale(value)
		corrector.SetSettings(settings)
	})
	scale.SetSelected(string(settings.Scale))

	amount := widget.NewLabel(fmt.Sprintf("Strength: %.0f%%", settings.Strength*100))
	strength := widget.NewSlider(0, 100)
	strength.Step = 1
	strength.SetValue(settings.Strength * 100)
	strength.OnChanged = func(value float64) {
		settings.Strength = value / 100
		corrector.SetSettings(settings)
		amount.SetText(fmt.Sprintf("Strength: %.0f%%", value))
	}

	return container.NewVBox(
		enabled,
		container.NewGridWithColumns(3,
			container.NewVBox(widget.NewLabel("Key"), key),
			container.NewVBox(widget.NewLabel("Scale"), scale),
			container.NewVBox(amount, strength),
		),
		widget.NewLabel("0% = original pitch · 100% = snap to scale · Minor = natural minor"),
	)
}
