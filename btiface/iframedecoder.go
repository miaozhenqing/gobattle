package btiface

type IFrameDecoder interface {
	Decode(b []byte) ([][]byte, error)
}
