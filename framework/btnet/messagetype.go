package btnet

type MessageType uint8

const (
	MessageRequest   MessageType = 1
	MessageResponse  MessageType = 2
	MessagePush      MessageType = 3
	MessageHandshake MessageType = 4
)
