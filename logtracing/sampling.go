package logtracing

import (
	"encoding/binary"
)

type Sampler func(SamplingParameters) bool

type SamplingParameters struct {
	ParentMeta spanMeta
	TraceID    TraceID
	SpanID     SpanID
	Name       string
}

func ProbabilitySampler(fraction float64) Sampler {
	if !(fraction >= 0) {
		fraction = 0
	} else if fraction >= 1 {
		return AlwaysSample()
	}

	traceIDUpperBound := uint64(fraction * (1 << 63))
	return Sampler(func(p SamplingParameters) bool {
		if p.ParentMeta.IsSampled {
			return true
		}
		x := binary.BigEndian.Uint64(p.TraceID[0:8]) >> 1
		return x < traceIDUpperBound
	})
}

func AlwaysSample() Sampler {
	return func(p SamplingParameters) bool {
		return true
	}
}

func NeverSample() Sampler {
	return func(p SamplingParameters) bool {
		return false
	}
}
