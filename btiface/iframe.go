package btiface

type IFrame interface {
	GetFlags() uint8
	GetBody() []byte
}
