package btiface

type IFrameDecoder interface {
	//Decode(b []byte) ([][]byte, error)

	Decode(b []byte) ([]IMessage, error)
}
