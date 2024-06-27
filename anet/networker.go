package anet

import (
	"abyss/and"
	"abyss/atype"
	"context"
	"errors"
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
	callq  chan PeerQueryCall
	ErrLog chan error
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
	result.ErrLog = make(chan error, 32)

	/////main worker/////

	AHMP_channel := make(chan AHMPReadRes, 64)

	accept_done := make(chan bool, 1)
	new_session_ch := make(chan *Session, 32)

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
		connect_fail_ch := make(chan ConnectFail, 32)
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

				result.ndh_lock.Lock()

				switch msg := ahmp_read.msg.(type) {
				case AHMPDisconnect:
					delete(result.peers, ahmp_read.peer.GetHash())
					ahmp_read.peer.Close()
					result.ErrRaise(msg.exitcode)
				case AHMPRaw_ID:
				case AHMPRaw_JN:
				case AHMPRaw_JOK:
				case AHMPRaw_JDN:
				case AHMPRaw_JNI:
				case AHMPRaw_MEM:
				case AHMPRaw_SNB:
				case AHMPRaw_CRR:
				case AHMPRaw_RST:
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
