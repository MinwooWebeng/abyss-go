package anet

import (
	"abyss/atype"
	"errors"

	"github.com/quic-go/quic-go"
)

type Session struct {
	connection  quic.Connection
	ahmp_stream quic.Stream //host control message protocol
	ahmp_parser AHMPParser
	identity    atype.AbyssIdentity
	address     atype.AbyssAddress
}

func NewSession(connection quic.Connection, ahmp_stream quic.Stream, ahmp_init_msg []byte) (*Session, error) {
	result := new(Session)

	result.connection = connection
	result.ahmp_stream = ahmp_stream

	ahmp_stream.Write(ahmp_init_msg)

	//TODO : ahmp parser
	init_message, err := result.ahmp_parser.Read(ahmp_stream)
	if err != nil {
		return nil, err
	}
	apd_id, ok := init_message.(AHMPRaw_ID)
	if !ok {
		return nil, errors.New("id exchange failed")
	}

	result.identity, err = atype.MakeAbyssIdentity(apd_id.pubkey, string(apd_id.name))
	if err != nil {
		return nil, err
	}

	result.address, ok = atype.MakeAbyssAddress2(result.identity.Hash, connection.RemoteAddr().String(), "")
	if !ok {
		return nil, errors.New("failed to parse remote address")
	}

	return result, nil
}

func (s *Session) GetHash() string {
	return s.identity.Hash
}
