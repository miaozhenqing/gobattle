package btiface

type IFrameDecoder interface {
	Decode(b []byte) ([]IMessage, error)
}
