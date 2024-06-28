package and

import (
	"strconv"
	"strings"
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
	eventType NeighborDiscoveryEventType
	localpath string                      //can be ""
	peer_hash string                      //never be nil
	peer      INeighborDiscoveryPeerBase  //can be nil
	path      string                      //can be ""
	world     INeighborDiscoveryWorldBase //can be nil
	status    int
	message   string
}

type INeighborDiscoveryHandler interface {
	ReserveEventListener(listener chan<- NeighborDiscoveryEvent)
	ReserveErrorListener(listener chan<- error)
	ReserveConnectCallback(func(address any))

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
}

// for testing purpose
func (e *NeighborDiscoveryEvent) Stringify() string {
	var sb strings.Builder
	switch e.eventType {
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
	sb.WriteString(e.localpath)
	sb.WriteString(",")
	sb.WriteString(e.peer_hash)
	sb.WriteString(",")
	sb.WriteString(e.path)
	if e.world != nil {
		sb.WriteString(",")
		sb.WriteString(e.world.GetUUID())
	}
	if e.status != 0 {
		sb.WriteString(",")
		sb.WriteString(strconv.Itoa(e.status))
		sb.WriteString(",")
		sb.WriteString(e.message)
	}
	return sb.String()
}
