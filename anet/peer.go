package anet

import (
	"abyss/and"
	"abyss/atype"
	"strconv"
	"strings"
	"sync/atomic"
)

type AHMPReadRes struct {
	peer *Peer
	msg  any
	err  error
}

type Peer struct {
	primary_session   *Session
	secondary_session *Session
	AhmpCh            chan AHMPReadRes
	open_session_n    atomic.Int32
}

func NewPeer(session *Session, ahmp_ch chan AHMPReadRes) *Peer {
	result := new(Peer)
	result.primary_session = session
	result.AhmpCh = ahmp_ch
	go func() {
		result.open_session_n.Add(1)
		for {
			msg, err := result.primary_session.ahmp_parser.Read(result.primary_session.ahmp_stream)
			if err != nil {
				_, ok := err.(*AHMPError)
				if !ok {
					if result.open_session_n.Add(-1) == 0 {
						result.AhmpCh <- AHMPReadRes{result, AHMPDisconnect{err}, nil}
					}
					return //should be channel/connection closed (may need revision)
				}
				result.AhmpCh <- AHMPReadRes{result, msg, err}
			}
		}
	}()

	return result
}

func (p *Peer) TryAddSession(Session *Session) bool {
	if p.primary_session.address != Session.address {
		return false
	}
	if p.secondary_session != nil {
		return false
	}
	p.secondary_session = Session
	go func() {
		p.open_session_n.Add(1)
		for {
			msg, err := p.secondary_session.ahmp_parser.Read(p.secondary_session.ahmp_stream)
			if err != nil {
				_, ok := err.(*AHMPError)
				if !ok {
					if p.open_session_n.Add(-1) == 0 {
						if p.open_session_n.Add(-1) == 0 {
							p.AhmpCh <- AHMPReadRes{p, AHMPDisconnect{err}, nil}
						}
					}
					return //should be channel/connection closed (may need revision)
				}
				p.AhmpCh <- AHMPReadRes{p, msg, err}
			}
		}
	}()
	return true
}
func (p *Peer) Close() {
	p.primary_session.connection.CloseWithError(0, "connection close")
	if p.secondary_session != nil {
		p.primary_session.connection.CloseWithError(0, "connection close")
	}
}

func (p *Peer) SendJN(path string) {
	p.primary_session.ahmp_stream.Write([]byte("AHMP/1.0 JN " + path + "\n\n"))
}
func (p *Peer) SendJOK(path string, world and.INeighborDiscoveryWorldBase) {
	body := world.GetJsonString()
	p.primary_session.ahmp_stream.Write([]byte("AHMP/1.0 JOK " + path + "\n"))
	p.primary_session.ahmp_stream.Write([]byte("Content-Length: " + strconv.Itoa(len(body)) + "\n\n"))
	p.primary_session.ahmp_stream.Write([]byte(body))
}
func (p *Peer) SendJDN(path string, status int, message string) {
	p.primary_session.ahmp_stream.Write([]byte("AHMP/1.0 JDN " + path + " " + strconv.Itoa(status) + " " + message + "\n\n"))
}
func (p *Peer) SendJNI(world and.INeighborDiscoveryWorldBase, member and.INeighborDiscoveryPeerBase) {
	address, _ := member.GetAddress().(atype.AbyssAddress)
	p.primary_session.ahmp_stream.Write([]byte("AHMP/1.0 JNI "))
	p.primary_session.ahmp_stream.Write(world.GetUUIDBytes())
	p.primary_session.ahmp_stream.Write([]byte(" " + address.Text + "\n\n"))
}
func (p *Peer) SendMEM(world and.INeighborDiscoveryWorldBase) {
	p.primary_session.ahmp_stream.Write([]byte("AHMP/1.0 MEM "))
	p.primary_session.ahmp_stream.Write(world.GetUUIDBytes())
	p.primary_session.ahmp_stream.Write([]byte("\n\n"))
}
func (p *Peer) SendSNB(world and.INeighborDiscoveryWorldBase, members_hash []string) {
	body := strings.Join(members_hash, ",")
	p.primary_session.ahmp_stream.Write([]byte("AHMP/1.0 SNB "))
	p.primary_session.ahmp_stream.Write(world.GetUUIDBytes())
	p.primary_session.ahmp_stream.Write([]byte("\nContent-Length: " + strconv.Itoa(len(body)) + "\n\n"))
	p.primary_session.ahmp_stream.Write([]byte(body))
}
func (p *Peer) SendCRR(world and.INeighborDiscoveryWorldBase, members_hash string) {
	p.primary_session.ahmp_stream.Write([]byte("AHMP/1.0 CRR "))
	p.primary_session.ahmp_stream.Write(world.GetUUIDBytes())
	p.primary_session.ahmp_stream.Write([]byte(" " + members_hash + "\n\n"))
}
func (p *Peer) SendRST(world_uuid string) {
	p.primary_session.ahmp_stream.Write([]byte("AHMP/1.0 RST " + world_uuid + "\n\n"))
}

func (p *Peer) GetAddress() any {
	return p.primary_session.address
}
func (p *Peer) GetHash() string {
	return p.primary_session.identity.Hash
}
