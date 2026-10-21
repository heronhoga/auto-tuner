package effects

import "math"

type chorus struct {
	buffer     []float64
	write      int
	sampleRate float64
	phase      [2]float64
	phaseStep  [2]float64
}

func newChorus(sampleRate float64) chorus {
	return chorus{
		buffer:     make([]float64, int(math.Ceil(0.030*sampleRate))+2),
		sampleRate: sampleRate,
		phase:      [2]float64{0, math.Pi / 2},
		phaseStep:  [2]float64{2 * math.Pi * 0.8 / sampleRate, 2 * math.Pi * 1.1 / sampleRate},
	}
}

func (c *chorus) process(input float64) float64 {
	c.buffer[c.write] = input
	// Two independently modulated voices create a chorus rather than a single
	// delayed copy. Fractional reads avoid stepping between delay samples.
	delayA := (0.018 + 0.004*math.Sin(c.phase[0])) * c.sampleRate
	delayB := (0.023 + 0.005*math.Sin(c.phase[1])) * c.sampleRate
	wet := 0.5 * (c.read(delayA) + c.read(delayB))
	for i := range c.phase {
		c.phase[i] = math.Mod(c.phase[i]+c.phaseStep[i], 2*math.Pi)
	}
	c.write = (c.write + 1) % len(c.buffer)
	return wet
}

func (c *chorus) read(delay float64) float64 {
	position := float64(c.write) - delay
	if position < 0 {
		position += float64(len(c.buffer))
	}
	left := int(position)
	fraction := position - float64(left)
	right := (left + 1) % len(c.buffer)
	return c.buffer[left]*(1-fraction) + c.buffer[right]*fraction
}
