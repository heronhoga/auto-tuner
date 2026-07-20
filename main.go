package main

import (
	"fmt"
	"os"
	"os/signal"

	utils "github.com/heronhoga/auto-tuner/utils/audio"
)

func main() {
    input, err := utils.NewInput()
    if err != nil {
        panic(err)
    }

    if err := input.Start(); err != nil {
        panic(err)
    }
    defer input.Stop()

    frames := input.Broadcaster.Subscribe()

    go func() {
        for frame := range frames {
            fmt.Printf(
                "Received %d samples (%d Hz, %d channels)\n",
                len(frame.Samples),
                input.Format.SampleRate,
                input.Format.Channels,
            )
        }
    }()

    fmt.Println("Listening...")

    sig := make(chan os.Signal, 1)
    signal.Notify(sig, os.Interrupt)
    <-sig
}