package btiface

type ISerializer interface {
	Serialize(msg IMessage) []byte
	Deserialize(bytes []byte) (IMessage, error)
}
