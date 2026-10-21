package yin

import (
	"math"
	"testing"
)

func TestFrequencyToNoteCents(t *testing.T) {
	for _, cents := range []float64{-35, 0, 25} {
		note := FrequencyToNote(440 * math.Exp2(cents/1200))
		if note.NoteName != "A" || note.Octave != 4 || math.Abs(note.Cents-cents) > 1e-8 {
			t.Errorf("offset %.1f cents: got %s%d %+.2f cents", cents, note.NoteName, note.Octave, note.Cents)
		}
	}
}
