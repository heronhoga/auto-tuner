package effects

import "math"

// A Schroeder-style reverb: parallel damped feedback delays build a decay,
// then serial all-pass filters diffuse it into a dense room tail.
type reverb struct {
	combs     []combFilter
	diffusers []allPassFilter
}

type combFilter struct {
	buffer   []float64
	position int
	feedback float64
	damping  float64
	filtered float64
}

type allPassFilter struct {
	buffer   []float64
	position int
}

func newReverb(sampleRate float64) reverb {
	var r reverb
	// Delays are tuned at 44.1 kHz and scaled to preserve their duration.
	for _, length := range []int{1116, 1188, 1277, 1356, 1422, 1491, 1557, 1617} {
		size := max(1, int(math.Round(float64(length)*sampleRate/44100)))
		r.combs = append(r.combs, combFilter{
			buffer:   make([]float64, size),
			feedback: math.Pow(0.001, float64(size)/sampleRate/1.4), // ~1.4 s low-frequency decay.
			damping:  math.Exp(-2 * math.Pi * 6000 / sampleRate),
		})
	}
	for _, length := range []int{556, 441, 341, 225} {
		size := max(1, int(math.Round(float64(length)*sampleRate/44100)))
		r.diffusers = append(r.diffusers, allPassFilter{buffer: make([]float64, size)})
	}
	return r
}

func (r *reverb) process(input float64) float64 {
	var wet float64
	for i := range r.combs {
		wet += r.combs[i].process(input * 0.3)
	}
	wet /= math.Sqrt(float64(len(r.combs)))
	for i := range r.diffusers {
		wet = r.diffusers[i].process(wet)
	}
	// Bound only the wet return; full-strength effects retain headroom while
	// zero-amount bypass remains sample-exact.
	return math.Tanh(wet)
}

func (c *combFilter) process(input float64) float64 {
	output := c.buffer[c.position]
	c.filtered = output*(1-c.damping) + c.filtered*c.damping
	c.buffer[c.position] = input + c.feedback*c.filtered
	c.position = (c.position + 1) % len(c.buffer)
	return output
}

func (a *allPassFilter) process(input float64) float64 {
	const feedback = 0.5
	delayed := a.buffer[a.position]
	value := input + feedback*delayed
	a.buffer[a.position] = value
	a.position = (a.position + 1) % len(a.buffer)
	return delayed - feedback*value
}
