package mqtt

type TopicFilter struct {
	Topic string
	QoS   uint8
}

// NOTE: second nibble of subscribe command is fixed to 0x2
type Subscribe struct {
	Version version

	PacketIdentifier uint16

	// Payload
	Topics []TopicFilter
}

type SubAck struct {
	Version version

	PacketIdentifier uint16

	ReturnCode uint8
}

type Unsubscribe struct {
	Version version

	PacketIdentifier uint16

	Topics []string
}

type UnsubAck struct {
	Version version

	PacketIdentifier uint16
}
