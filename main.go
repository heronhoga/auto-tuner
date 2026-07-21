package main

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/heronhoga/auto-tuner/ui/oscilloscope"
	utils "github.com/heronhoga/auto-tuner/utils/audio"
)

func main() {
    input, err := utils.NewInput()
    if err != nil {
        panic(err)
    }

    model := oscilloscope.NewModel()

    scope := oscilloscope.New(model)

    if err := input.Start(); err != nil {
        panic(err)
    }
    defer input.Stop()

    frames := input.Broadcaster.Subscribe()

    go func() {
        for frame := range frames {
            model.Update(frame.Samples)
        }
    }()

    a := app.New()
    w := a.NewWindow("Auto Tuner")

    w.SetContent(scope)
    w.Resize(fyne.NewSize(900, 300))

    go func() {
        ticker := time.NewTicker(time.Second / 60)
        defer ticker.Stop()

        for range ticker.C {
            fyne.Do(func() {
                scope.Refresh()
            })
        }
    }()

    w.ShowAndRun()
}