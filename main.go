package main

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"github.com/heronhoga/auto-tuner/ui"
	"github.com/heronhoga/auto-tuner/ui/oscilloscope"
	utils "github.com/heronhoga/auto-tuner/utils/audio"
	"github.com/heronhoga/auto-tuner/utils/effects"
	"github.com/heronhoga/auto-tuner/utils/tune"
	"github.com/heronhoga/auto-tuner/utils/yin"
)

func main() {
	// input
	input, err := utils.NewInput()
	if err != nil {
		panic(err)
	}
	defer input.Stop()

	// output playback
	playbackBuffer := utils.NewPlaybackBuffer(48000)
	output, err := utils.NewOutput(playbackBuffer)
	if err != nil {
		panic(err)
	}
	defer output.Stop()

	if err := output.Start(); err != nil {
		panic(err)
	}

	// analytics
	ringBuffer := utils.NewRingBuffer(4096)
	scope := oscilloscope.New(ringBuffer)

	detector := yin.New(48000)
	detector.Threshold = 0.10
	corrector := tune.NewAutoCorrector(48000)
	fx := effects.NewProcessor(48000)
	frames := input.Broadcaster.Subscribe()

	if err := input.Start(); err != nil {
		panic(err)
	}

	go func() {
		var corrected []float32
		for frame := range frames {
			// write samples to ring buffer
			ringBuffer.Write(frame.Samples)

			// Correct pitch, then apply chorus and reverb before playback.
			if cap(corrected) < len(frame.Samples) {
				corrected = make([]float32, len(frame.Samples))
			}
			corrected = corrected[:len(frame.Samples)]
			corrector.Process(frame.Samples, corrected)
			fx.Process(corrected, corrected)
			playbackBuffer.Write(corrected)
		}

	}()

	// UI
	a := app.New()
	w := a.NewWindow("Hoga Auto Tuner")

	controls := container.NewVBox(
		ui.NewAudioDevices(input, output),
		ui.NewPitchCorrection(corrector),
		ui.NewEffects(fx),
	)
	w.SetContent(container.NewBorder(controls, nil, nil, nil, scope))
	w.Resize(fyne.NewSize(900, 650))

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
