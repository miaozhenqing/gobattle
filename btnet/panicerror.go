package btnet

import "fmt"

type PanicError struct {
	Value interface{}
	Stack []byte
}

func (e *PanicError) Error() string {
	return fmt.Sprintf("handler panic: %v", e.Value)
}
