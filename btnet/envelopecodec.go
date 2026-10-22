package btnet

import "gobattle/btiface"

var (
	_ btiface.IInHandler  = (*EnvelopeCodec)(nil)
	_ btiface.IOutHandler = (*EnvelopeCodec)(nil)
)

type EnvelopeCodec struct {
}

func NewEnvelopeCodec() *EnvelopeCodec {
	return &EnvelopeCodec{}
}

func (e *EnvelopeCodec) OnRead(ctx btiface.IHandlerContext, msg interface{}) {
	f, ok := msg.(btiface.IFrame)
	if !ok {
		ctx.FireRead(msg)
		return
	}
	env, err := decodeEnvelope(f.GetBody())
	if err != nil {
		ctx.FireException(err)
		return
	}
	ctx.FireRead(env)
}

func (e *EnvelopeCodec) OnWrite(ctx btiface.IHandlerContext, msg interface{}) {
	env, ok := msg.(*btiface.Envelope)
	if !ok {
		ctx.FireWrite(msg)
		return
	}
	ctx.FireWrite(NewFrame(0, encodeEnvelope(env)))
}
