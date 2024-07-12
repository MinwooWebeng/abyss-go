package aresource

import "io"

type IServiceProvider interface {
	AllowToken(string)
	CeaseToken(string)

	Instantiate(string) IServiceInstance
}

type IServiceInstance interface {
	io.ReadWriter
	AttachTransmissionResource(func([]byte) error, func() ([]byte, error)) //datagram Tx/Rx
}
