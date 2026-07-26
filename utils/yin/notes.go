package yin

import (
	"fmt"
	"math"
)

var noteNames = []string{
	"C", "C#", "D", "D#", "E", "F",
	"F#", "G", "G#", "A", "A#", "B",
}

type NoteResult struct {
	NoteName string
	Octave int
	Frequency float64
	Cents float64
	MIDI int
}

func FrequencyToNote(freq float64) NoteResult {
	if freq <= 0 {
		return NoteResult{}
	}

	midiFloat := 69 + 12*math.Log2(freq/440.0)
	midi := int(math.Round(midiFloat))

	noteIndex := midi % 12
	if noteIndex < 0 {
		noteIndex += 12
	}

	octave := (midi / 12) - 1
	cents := 1200 * (midiFloat - float64(midi))

	return NoteResult{
		NoteName: noteNames[noteIndex],
		Octave: octave,
		Frequency: freq,
		Cents: cents,
		MIDI: midi,

	}
}

func (n NoteResult) String() string {
    return fmt.Sprintf("%s%d (%+.2f cents)", n.NoteName, n.Octave, n.Cents)
}