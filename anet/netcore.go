package anet

import (
	"abyss/atype"
)

type INetCore interface {
	//blocking calls
	Connect(atype.AbyssAddress) (*Session, error)
	Accept() (*Session, error)

	LocalIdentity() atype.AbyssIdentity
	LocalAddr() atype.AbyssAddress

	Close()
}
