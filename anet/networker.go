package anet

import (
	"abyss/and"
	"abyss/atype"
	"context"
	"errors"
	"strings"
	"sync"
)

type PeerQueryReturn struct {
	result *Peer
	err    error
}

type PeerQueryCall struct {
	ret_ch chan PeerQueryReturn
	arg    any //string or atype.AbyssAddress
}

type ConnectFail struct {
	hash string
	err  error
}

type Networker struct {
	//internal thread access only
	netcore      INetCore
	ndh          and.INeighborDiscoveryHandler
	ndh_lock     sync.Mutex
	peers        map[string]*Peer
	ongoing_dial map[string][]chan PeerQueryReturn

	fin_wg sync.WaitGroup

	//access from external thread
	callq      chan PeerQueryCall
	NdhEventCh chan and.NeighborDiscoveryEvent
	ErrLog     chan error
}

func (n *Networker) ErrRaise(err error) {
	select {
	case n.ErrLog <- err:
	default:
	}
}

func CreateNetworker(pubkey_pem []byte, name string) (*Networker, error) {
	result := new(Networker)
	id, err := atype.MakeAbyssIdentity(pubkey_pem, name)
	if err != nil {
		return nil, err
	}

	result.netcore, err = NewGoQuicNetCore(id)
	if err != nil {
		return nil, err
	}

	result.ndh = and.NewNeighborDiscoveryHandler(id.Hash)

	result.peers = make(map[string]*Peer)
	result.ongoing_dial = make(map[string][]chan PeerQueryReturn)

	result.callq = make(chan PeerQueryCall, 32)
	result.NdhEventCh = make(chan and.NeighborDiscoveryEvent, 64)
	result.ErrLog = make(chan error, 32)

	result.ndh.ReserveEventListener(result.NdhEventCh)
	result.ndh.ReserveErrorListener(result.ErrLog)

	/////main worker/////

	AHMP_channel := make(chan AHMPReadRes, 64)

	accept_done := make(chan bool, 1)
	new_session_ch := make(chan *Session, 32)
	connect_fail_ch := make(chan ConnectFail, 32)

	ConnectAsync := func(_address any) {
		address, _ := _address.(atype.AbyssAddress)
		go func() {
			session, err := result.netcore.Connect(address)
			if err != nil {
				connect_fail_ch <- ConnectFail{address.Pubkey_hash, err}
			} else {
				new_session_ch <- session
			}
		}()
	}
	result.ndh.ReserveConnectCallback(ConnectAsync)

	result.fin_wg.Add(1)
	go func() { //Accepter
		defer result.fin_wg.Done()
		for {
			session, err := result.netcore.Accept()
			if err != nil {
				if err == context.Canceled {
					accept_done <- true
					return
				}
				result.ErrRaise(err)
			} else {
				new_session_ch <- session
			}
		}
	}()

	result.fin_wg.Add(1)
	go func() { //Peer query/connect handler
		defer result.fin_wg.Done()
		for {
			select {
			case query_call := <-result.callq:
				var hash string
				switch ct := query_call.arg.(type) {
				case string:
					hash = ct
				case atype.AbyssAddress:
					hash = ct.Pubkey_hash
				}

				peer, ok := result.peers[hash]
				if ok { //peer found
					query_call.ret_ch <- PeerQueryReturn{peer, nil}
					break
				}

				//no peer found
				waiting_calls, ok := result.ongoing_dial[hash]
				if ok { //there is ongoing dial. append waiting call.
					waiting_calls = append(waiting_calls, query_call.ret_ch)
					result.ongoing_dial[hash] = waiting_calls
					break
				}

				//no ongoing dial
				switch ct := query_call.arg.(type) {
				case string:
					query_call.ret_ch <- PeerQueryReturn{nil, errors.New("peer not found")}
				case atype.AbyssAddress:
					result.ongoing_dial[hash] = []chan PeerQueryReturn{query_call.ret_ch}
					go func() {
						session, err := result.netcore.Connect(ct)
						if err != nil {
							connect_fail_ch <- ConnectFail{ct.Pubkey_hash, err}
						} else {
							new_session_ch <- session
						}
					}()
				}
			case new_session := <-new_session_ch:
				peer, ok := result.peers[new_session.GetHash()]
				if ok { //session already exists, duplicate connection
					if !peer.TryAddSession(new_session) {
						//triple connection
						result.ErrRaise(errors.New("triple session"))
						new_session.connection.CloseWithError(409, "triple session")
					}
				}

				//new peer
				peer = NewPeer(new_session, AHMP_channel)
				result.peers[new_session.GetHash()] = peer

				result.ndh_lock.Lock()
				result.ndh.Connected(peer)
				result.ndh_lock.Unlock()

				ret_list, ok := result.ongoing_dial[new_session.GetHash()]
				if ok { //there was ongoing dial
					for _, ret_ch := range ret_list {
						ret_ch <- PeerQueryReturn{peer, nil}
					}
					delete(result.ongoing_dial, new_session.GetHash())
				}
			case conn_fail := <-connect_fail_ch:
				ret_list := result.ongoing_dial[conn_fail.hash]
				for _, ret_ch := range ret_list {
					ret_ch <- PeerQueryReturn{nil, conn_fail.err}
				}
				delete(result.ongoing_dial, conn_fail.hash)
			case ahmp_read := <-AHMP_channel:
				if ahmp_read.err != nil {
					result.ErrRaise(ahmp_read.err)
					break
				}

				switch msg := ahmp_read.msg.(type) {
				case AHMPExit:
					ahmp_read.peer.Close()
					delete(result.peers, ahmp_read.peer.GetHash())

					result.ndh_lock.Lock()
					result.ndh.Disconnected(ahmp_read.peer.GetHash())
					result.ndh_lock.Unlock()

					result.ErrRaise(msg.exitcode)
				case AHMPRaw_ID:
					result.ErrRaise(errors.New("duplicate AHMP ID"))
				case AHMPRaw_JN:
					result.ndh_lock.Lock()
					result.ndh.OnJN(ahmp_read.peer, string(msg.path))
					result.ndh_lock.Unlock()
				case AHMPRaw_JOK:
					world, err := ParseWorldJson(msg.world)
					if err != nil {
						ahmp_read.peer.Signal(err)
						break
					}

					result.ndh_lock.Lock()
					result.ndh.OnJOK(ahmp_read.peer, string(msg.path), world)
					result.ndh_lock.Unlock()
				case AHMPRaw_JDN:
					result.ndh_lock.Lock()
					result.ndh.OnJDN(ahmp_read.peer, string(msg.path), msg.status, string(msg.message))
					result.ndh_lock.Unlock()
				case AHMPRaw_JNI:
					joiner_address, ok := atype.ParseAbyssAddress(string(msg.address))
					if !ok {
						ahmp_read.peer.Signal(errors.New("ahmp corrupted"))
						break
					}

					result.ndh_lock.Lock()
					result.ndh.OnJNI(ahmp_read.peer, string(msg.world_uuid), joiner_address, joiner_address.Pubkey_hash)
					result.ndh_lock.Unlock()
				case AHMPRaw_MEM:
					result.ndh_lock.Lock()
					result.ndh.OnMEM(ahmp_read.peer, string(msg.world_uuid))
					result.ndh_lock.Unlock()
				case AHMPRaw_SNB:
					split := strings.Split(string(msg.members_hash), ",")

					result.ndh_lock.Lock()
					result.ndh.OnSNB(ahmp_read.peer, string(msg.world_uuid), split)
					result.ndh_lock.Unlock()
				case AHMPRaw_CRR:
					result.ndh_lock.Lock()
					result.ndh.OnCRR(ahmp_read.peer, string(msg.world_uuid), string(msg.missing_hash))
					result.ndh_lock.Unlock()
				case AHMPRaw_RST:
					result.ndh_lock.Lock()
					result.ndh.OnRST(ahmp_read.peer, string(msg.world_uuid))
					result.ndh_lock.Unlock()
				default:
					result.ErrRaise(errors.New("unknown message type"))
				}

				result.ndh_lock.Unlock()

			case <-accept_done:
				//TODO: disconnect all
				close(result.ErrLog)
				return
			}
		}
	}()

	return result, nil
}

func (n *Networker) WaitClose() {
	n.netcore.Close()
	n.fin_wg.Wait()
}

// this may take long - connect
func (n *Networker) GetPeerByAddress(address atype.AbyssAddress) (*Peer, error) {
	return_ch := make(chan PeerQueryReturn)
	n.callq <- PeerQueryCall{return_ch, address}

	result := <-return_ch
	return result.result, result.err
}

func (n *Networker) GetPeerByHash(hash string) (*Peer, error) {
	return_ch := make(chan PeerQueryReturn)
	n.callq <- PeerQueryCall{return_ch, hash}

	result := <-return_ch
	return result.result, result.err
}

func (n *Networker) OpenWorld(path string, world *World) bool {
	n.ndh_lock.Lock()
	defer n.ndh_lock.Unlock()
	return n.ndh.OpenWorld(path, world)
}
func (n *Networker) CloseWorld(path string) {
	n.ndh_lock.Lock()
	defer n.ndh_lock.Unlock()
	n.ndh.CloseWorld(path)
}
func (n *Networker) ChangeWorldPath(prev_path string, new_path string) bool {
	n.ndh_lock.Lock()
	defer n.ndh_lock.Unlock()
	return n.ndh.ChangeWorldPath(prev_path, new_path)
}
func (n *Networker) GetWorld(path string) (*World, bool) {
	n.ndh_lock.Lock()
	defer n.ndh_lock.Unlock()
	world, ok := n.ndh.GetWorld(path)
	result, _ := world.(*World)
	return result, ok
}

func (n *Networker) JoinConnected(local_path string, peer *Peer, path string) {
	n.ndh_lock.Lock()
	defer n.ndh_lock.Unlock()
	n.ndh.JoinConnected(local_path, peer, path)
}
func (n *Networker) JoinAny(local_path string, address any, peer_hash string, path string) {
	n.ndh_lock.Lock()
	defer n.ndh_lock.Unlock()
	n.ndh.JoinAny(local_path, address, peer_hash, path)
}
