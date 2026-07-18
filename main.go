package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/heronhoga/auto-tuner/utils"
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

	go func() {
		for frame := range input.Frames {
			fmt.Printf("Received %d samples, (%d Hz, %d channels)\n", len(frame.Samples), frame.SampleRate, frame.Channels)
		}
	} ()

	fmt.Println("Listening..")
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig

}