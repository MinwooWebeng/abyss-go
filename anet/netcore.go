package anet

import (
	"abyss/atype"
)

type INetCore interface {
	//blocking calls
	Connect(atype.AbyssAddress) (*Transmission, error)
	Accept() (*Transmission, error)

	LocalIdentity() atype.AbyssIdentity
	LocalAddr() atype.AbyssAddress

	Close()
}
