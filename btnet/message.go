package btnet

type Message struct {
	msgId uint32
	data  []byte
}

func NewMessage(msgId uint32, bytes []byte) *Message {
	return &Message{
		msgId: msgId,
		data:  bytes,
	}
}

func (msg *Message) GetDataLen() uint32 {
	return uint32(len(msg.data))
}

func (msg *Message) GetMsgId() uint32 {
	return msg.msgId
}

func (msg *Message) GetData() []byte {
	return msg.data
}
