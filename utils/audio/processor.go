package utils

type Processor interface {
	Process(input []float32, output []float32)
}

type PassthroughProcessor struct {
}

func NewPassthroughProcessor() *PassthroughProcessor {
	return &PassthroughProcessor{}
}

func (p *PassthroughProcessor) Process(input []float32, output []float32) {
	copy(output, input)
}

