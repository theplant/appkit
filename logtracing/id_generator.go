package logtracing

import (
	crand "crypto/rand"
	"encoding/binary"
	"math/rand"
	"sync"
)

type IDGenerator interface {
	NewTraceID() TraceID
	NewSpanID() SpanID
}

type randomIDGenerator struct {
	sync.Mutex
	randSource *rand.Rand
}

var _ IDGenerator = &randomIDGenerator{}

func (gen *randomIDGenerator) NewTraceID() TraceID {
	gen.Lock()
	defer gen.Unlock()
	tid := TraceID{}
	gen.randSource.Read(tid[:])
	return tid
}

func (gen *randomIDGenerator) NewSpanID() SpanID {
	gen.Lock()
	defer gen.Unlock()
	sid := SpanID{}
	gen.randSource.Read(sid[:])
	return sid
}

func defaultIDGenerator() IDGenerator {
	gen := &randomIDGenerator{}
	var rngSeed int64
	_ = binary.Read(crand.Reader, binary.LittleEndian, &rngSeed)
	gen.randSource = rand.New(rand.NewSource(rngSeed))
	return gen
}
