package and

import (
	"strconv"
	"strings"
	"time"
)

type NeighborDiscoveryEventType int

const (
	JoinDenied  NeighborDiscoveryEventType = iota
	JoinExpired NeighborDiscoveryEventType = iota
	JoinSuccess NeighborDiscoveryEventType = iota
	PeerJoin    NeighborDiscoveryEventType = iota
	PeerLeave   NeighborDiscoveryEventType = iota
)

type NeighborDiscoveryEvent struct {
	EventType NeighborDiscoveryEventType
	Localpath string                      //can be ""
	Peer_hash string                      //never be nil
	Peer      INeighborDiscoveryPeerBase  //can be nil
	Path      string                      //can be ""
	World     INeighborDiscoveryWorldBase //can be nil
	Status    int
	Message   string
}

type INeighborDiscoveryHandler interface {
	ReserveEventListener(listener chan<- NeighborDiscoveryEvent)
	ReserveErrorListener(listener chan<- error)
	ReserveConnectCallback(func(address any))
	ReserveSNBTimer(func(time.Duration, string))

	OpenWorld(path string, world INeighborDiscoveryWorldBase) bool
	CloseWorld(path string)
	ChangeWorldPath(prev_path string, new_path string) bool
	GetWorld(path string) (INeighborDiscoveryWorldBase, bool)

	Connected(peer INeighborDiscoveryPeerBase)
	Disconnected(peer_hash string) //also connect fail.
	JoinConnected(local_path string, peer INeighborDiscoveryPeerBase, path string)
	JoinAny(local_path string, address any, peer_hash string, path string)
	OnJN(peer INeighborDiscoveryPeerBase, path string)
	OnJOK(peer INeighborDiscoveryPeerBase, path string, world INeighborDiscoveryWorldBase)
	OnJDN(peer INeighborDiscoveryPeerBase, path string, status int, message string)
	OnJNI(peer INeighborDiscoveryPeerBase, world_uuid string, address any, joiner_hash string)
	OnMEM(peer INeighborDiscoveryPeerBase, world_uuid string)
	OnSNB(peer INeighborDiscoveryPeerBase, world_uuid string, members_hash []string)
	OnCRR(peer INeighborDiscoveryPeerBase, world_uuid string, missing_member_hash string)
	OnRST(peer INeighborDiscoveryPeerBase, world_uuid string)
	OnWorldErr(peer INeighborDiscoveryPeerBase, world_uuid string)
	OnSNBTimeout(world_uuid string)
}

// for testing purpose
func (e *NeighborDiscoveryEvent) Stringify() string {
	var sb strings.Builder
	switch e.EventType {
	case JoinDenied:
		sb.WriteString("JoinDenied ")
	case JoinExpired:
		sb.WriteString("JoinExpired ")
	case JoinSuccess:
		sb.WriteString("JoinSuccess ")
	case PeerJoin:
		sb.WriteString("PeerJoin ")
	case PeerLeave:
		sb.WriteString("PeerLeave ")
	}
	sb.WriteString(e.Localpath)
	sb.WriteString(",")
	sb.WriteString(e.Peer_hash)
	sb.WriteString(",")
	sb.WriteString(e.Path)
	if e.World != nil {
		sb.WriteString(",")
		sb.WriteString(e.World.GetUUID())
	}
	if e.Status != 0 {
		sb.WriteString(",")
		sb.WriteString(strconv.Itoa(e.Status))
		sb.WriteString(",")
		sb.WriteString(e.Message)
	}
	return sb.String()
}
