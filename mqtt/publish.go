package mqtt

type Publish struct {
	Version version

	// Flags
	Duplicate bool
	QoSLevel  uint16
	Retain    bool

	// Variable header
	TopicName        string
	PacketIdentifier uint16

	Payload []byte
}

type PubAck struct {
	Version version

	// Variable header
	PacketIdentifier uint16
}

type PubRec struct {
	Version version

	// Variable header
	PacketIdentifier uint16
}

type PubRel struct {
	Version version

	// Variable header
	PacketIdentifier uint16
}

type PubComp struct {
	Version version

	// Variable header
	PacketIdentifier uint16
}
