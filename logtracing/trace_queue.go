package logtracing

func QueueConsumerKVs() []interface{} {
	return []interface{}{
		"span.type", "queue",
		"span.role", "consumer",
	}
}

func QueueProducerKVs() []interface{} {
	return []interface{}{
		"span.type", "queue",
		"span.role", "producer",
	}
}
