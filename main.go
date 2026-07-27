package main

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/heronhoga/auto-tuner/ui/oscilloscope"
	utils "github.com/heronhoga/auto-tuner/utils/audio"
	"github.com/heronhoga/auto-tuner/utils/yin"
)

func main() {
    input, err := utils.NewInput()
    if err != nil {
        panic(err)
    }

    ringBuffer := utils.NewRingBuffer(4096)
    scope := oscilloscope.New(ringBuffer)

	detector := yin.New(48000)
	detector.Threshold = 0.10

    if err := input.Start(); err != nil {
        panic(err)
    }
    defer input.Stop()

    frames := input.Broadcaster.Subscribe()

    go func() {
        for frame := range frames {
            ringBuffer.Write(frame.Samples)
        }
    }()

    a := app.New()
    w := a.NewWindow("Hoga Auto Tuner")

    w.SetContent(scope)
    w.Resize(fyne.NewSize(900, 300))

    go func() {
        ticker := time.NewTicker(time.Second / 60)
        defer ticker.Stop()

        for range ticker.C {
            // oscilloscope
            fyne.Do(func() {
                scope.Refresh()
            })

            samples := ringBuffer.ReadLatest(4096)
            if len(samples) < 4096 {
                continue
            }

            result, err := detector.Detect(samples)
            if err != nil {
                continue
            }

            fyne.Do(func() {
                scope.SetNoteResult(result)
            })
            fmt.Printf("freq=%.2f Hz note=%s%d cents=%+.2f\n",
				result.Frequency,
				result.NoteName,
				result.Octave,
				result.Cents,
			)

        }
    }()

    w.ShowAndRun()
}