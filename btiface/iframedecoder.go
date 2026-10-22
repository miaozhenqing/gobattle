package btiface

type IFrameDecoder interface {
	Decode(b []byte) ([]IFrame, error)
}
