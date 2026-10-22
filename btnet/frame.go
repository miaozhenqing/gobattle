package btnet

type Frame struct {
	flags uint8
	body  []byte
}

func NewFrame(flags uint8, body []byte) *Frame {
	return &Frame{
		flags: flags,
		body:  body,
	}
}

func (f *Frame) GetFlags() uint8 {
	return f.flags
}

func (f *Frame) GetBody() []byte {
	return f.body
}
