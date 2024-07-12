package and

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"
)

type NeighborDiscoveryTestWorld struct {
	UUID string
}

func (w *NeighborDiscoveryTestWorld) GetUUID() string {
	return w.UUID
}
func (w *NeighborDiscoveryTestWorld) GetUUIDBytes() []byte {
	return []byte(w.GetUUID())
}
func (w *NeighborDiscoveryTestWorld) GetJsonBytes() []byte {
	return []byte("{UUID:\"" + w.UUID + "\"}")
}

var world_uuid_counter int = 3001

func NewWorld_Testimpl() *NeighborDiscoveryTestWorld {
	result := new(NeighborDiscoveryTestWorld)
	result.UUID = "world-uuid-" + strconv.Itoa(world_uuid_counter)
	world_uuid_counter++
	return result
}

type NeighborDiscoveryTestPeer struct {
	hash    string
	address any

	_log chan string
}

var peer_uuid_counter int = 70001

func NewNeighborDiscoveryTestPeer() *NeighborDiscoveryTestPeer {
	result := new(NeighborDiscoveryTestPeer)
	result.hash = "peer-uid-" + strconv.Itoa(peer_uuid_counter)
	peer_uuid_counter++
	result.address = "noaddr"
	result._log = make(chan string, 30)
	return result
}

func (p *NeighborDiscoveryTestPeer) Log(s string) {
	p._log <- s
}

func (p *NeighborDiscoveryTestPeer) SendJN(path string) {
	p.Log("AHMP/1.0 JN " + path)
}
func (p *NeighborDiscoveryTestPeer) SendJOK(path string, w INeighborDiscoveryWorldBase) {
	var world = w.GetJsonBytes()
	p.Log("AHMP/1.0 JOK " + path + " 200 OK\n" +
		"Content-Length: " + strconv.Itoa(len(world)) + "\n" +
		"\n" +
		string(world))
}
func (p *NeighborDiscoveryTestPeer) SendJDN(path string, status int, msg string) {
	p.Log("AHMP/1.0 JDN " + path + " " + strconv.Itoa(status) + " " + msg)
}
func (p *NeighborDiscoveryTestPeer) SendJNI(w INeighborDiscoveryWorldBase, j INeighborDiscoveryPeerBase) {
	var joiner = j.GetHash()
	p.Log("AHMP/1.0 JNI " + w.GetUUID() + "\n" +
		"Content-Length: " + strconv.Itoa(len(joiner)) + "\n" +
		"\n" +
		joiner)
}
func (p *NeighborDiscoveryTestPeer) SendMEM(w INeighborDiscoveryWorldBase) {
	p.Log("AHMP/1.0 MEM " + w.GetUUID())
}
func (p *NeighborDiscoveryTestPeer) SendSNB(w INeighborDiscoveryWorldBase, i []string) {
	var snb_sb strings.Builder
	snb_sb.WriteString("[")
	for c := 0; c < len(i)-1; c++ {
		snb_sb.WriteString(i[c])
		snb_sb.WriteString(",")
	}
	snb_sb.WriteString(i[len(i)-1])
	snb_sb.WriteString("]")

	var sb strings.Builder
	sb.WriteString("AHMP/1.0 SNB " + w.GetUUID() + "\n")
	sb.WriteString("Content-Length: " + strconv.Itoa(snb_sb.Len()) + "\n\n")
	sb.WriteString(snb_sb.String())
	p.Log(sb.String())
}
func (p *NeighborDiscoveryTestPeer) SendCRR(w INeighborDiscoveryWorldBase, i string) {
	var crr_sb strings.Builder
	crr_sb.WriteString(i)

	var sb strings.Builder
	sb.WriteString("AHMP/1.0 CRR " + w.GetUUID() + "\n")
	sb.WriteString("Content-Length: " + strconv.Itoa(crr_sb.Len()) + "\n\n")
	sb.WriteString(crr_sb.String())
	p.Log(sb.String())
}
func (p *NeighborDiscoveryTestPeer) SendRST(world_uuid string) {
	p.Log("AHMP/1.0 RST " + world_uuid)
}
func (p *NeighborDiscoveryTestPeer) GetHash() string {
	return p.hash
}
func (p *NeighborDiscoveryTestPeer) GetAddress() any {
	return p.address
}

type LocalHost struct {
	local_peer *NeighborDiscoveryTestPeer
	ndh        INeighborDiscoveryHandler
}

func NewLocalHost() *LocalHost {
	local_host := NewNeighborDiscoveryTestPeer()
	ndh := NewNeighborDiscoveryHandler("local_host_hash")
	ndh.ReserveConnectCallback(func(address any) {
		local_host.Log("connect")
	})
	ndh.ReserveSNBTimer(func(time.Duration, string) {})
	event_ch := make(chan NeighborDiscoveryEvent, 1)
	go func() {
		for {
			event := <-event_ch
			local_host.Log(event.Stringify())
		}
	}()
	ndh.ReserveEventListener(event_ch)
	error_ch := make(chan error, 1)
	go func() {
		for {
			event := <-error_ch
			local_host.Log(event.Error())
		}
	}()
	ndh.ReserveErrorListener(error_ch)
	return &LocalHost{local_host, ndh}
}

func TestOpenWorld(t *testing.T) {
	local_host := NewLocalHost()
	ndh := local_host.ndh
	if !ndh.OpenWorld("/", NewWorld_Testimpl()) {
		t.Fail()
	}
	ndh.CloseWorld("/")

	for len(local_host.local_peer._log) > 0 {
		fmt.Println(<-local_host.local_peer._log)
	}
}
func TestJoin1(t *testing.T) {
	local_host := NewLocalHost()
	ndh := local_host.ndh

	join_target := NewNeighborDiscoveryTestPeer()
	join_world := NewWorld_Testimpl()
	ndh.JoinAny("/", "*", join_target.GetHash(), "/target")
	ndh.Connected(join_target)
	ndh.OnJOK(join_target, "/target", join_world)

	time.Sleep(time.Second)
	for len(local_host.local_peer._log) > 0 {
		fmt.Println(<-local_host.local_peer._log)
	}
	fmt.Println("<<<join target" + join_target.GetHash() + ">>>")
	for len(join_target._log) > 0 {
		fmt.Println(<-join_target._log)
	}
}
func TestJoin2(t *testing.T) {
	local_host := NewLocalHost()
	ndh := local_host.ndh

	join_target := NewNeighborDiscoveryTestPeer()
	join_world := NewWorld_Testimpl()
	ndh.Connected(join_target)
	ndh.JoinConnected("/", join_target, "/target")
	ndh.OnJOK(join_target, "/target", join_world)

	time.Sleep(time.Second)
	for len(local_host.local_peer._log) > 0 {
		fmt.Println(<-local_host.local_peer._log)
	}
	fmt.Println("<<<join target" + join_target.GetHash() + ">>>")
	for len(join_target._log) > 0 {
		fmt.Println(<-join_target._log)
	}
}
func TestJoin3(t *testing.T) {
	local_host := NewLocalHost()
	ndh := local_host.ndh

	join_target := NewNeighborDiscoveryTestPeer()
	join_world := NewWorld_Testimpl()
	ndh.Connected(join_target)
	ndh.JoinAny("/", "*", join_target.GetHash(), "/target")
	ndh.OnJOK(join_target, "/target", join_world)

	time.Sleep(time.Second)
	for len(local_host.local_peer._log) > 0 {
		fmt.Println(<-local_host.local_peer._log)
	}
	fmt.Println("<<<join target" + join_target.GetHash() + ">>>")
	for len(join_target._log) > 0 {
		fmt.Println(<-join_target._log)
	}
}
func TestMEM(t *testing.T) {
	local_host := NewLocalHost()
	ndh := local_host.ndh

	peer_target := NewNeighborDiscoveryTestPeer()
	world := NewWorld_Testimpl()

	if !ndh.OpenWorld("/default", world) {
		t.Error("failed to open world")
	}
	ndh.Connected(peer_target)
	ndh.OnMEM(peer_target, world.GetUUID())

	time.Sleep(time.Second)
	for len(local_host.local_peer._log) > 0 {
		fmt.Println(<-local_host.local_peer._log)
	}
	fmt.Println("<<<join target" + peer_target.GetHash() + ">>>")
	for len(peer_target._log) > 0 {
		fmt.Println(<-peer_target._log)
	}
}

func TestPrematureJoin(t *testing.T) {
	local_host := NewLocalHost()
	ndh := local_host.ndh

	peer_target := NewNeighborDiscoveryTestPeer()
	world := NewWorld_Testimpl()

	peer_third := NewNeighborDiscoveryTestPeer()

	ndh.JoinAny("/", "noaddr", peer_target.GetHash(), "/w")

	ndh.Connected(peer_third)
	ndh.OnMEM(peer_third, world.GetUUID())

	ndh.Connected(peer_target)
	ndh.OnJOK(peer_target, "/w", world)

	time.Sleep(time.Second)
	for len(local_host.local_peer._log) > 0 {
		fmt.Println(<-local_host.local_peer._log)
	}
	fmt.Println("<<<join target" + peer_target.GetHash() + ">>>")
	for len(peer_target._log) > 0 {
		fmt.Println(<-peer_target._log)
	}
}

func TestJoinFail1(t *testing.T) {
	local_host := NewLocalHost()
	ndh := local_host.ndh

	peer_target := NewNeighborDiscoveryTestPeer()
	world := NewWorld_Testimpl()

	peer_third := NewNeighborDiscoveryTestPeer()

	ndh.JoinAny("/", "noaddr", peer_target.GetHash(), "/w")

	ndh.Connected(peer_third)
	ndh.OnMEM(peer_third, world.GetUUID())

	ndh.Connected(peer_target)
	ndh.OnJDN(peer_target, "/w", 404, "Not Found")

	time.Sleep(time.Second)
	for len(local_host.local_peer._log) > 0 {
		fmt.Println(<-local_host.local_peer._log)
	}
	fmt.Println("<<<join target" + peer_target.GetHash() + ">>>")
	for len(peer_target._log) > 0 {
		fmt.Println(<-peer_target._log)
	}
	fmt.Println("<<<third peer" + peer_third.GetHash() + ">>>")
	for len(peer_third._log) > 0 {
		fmt.Println(<-peer_third._log)
	}
}

func TestJoinFail2(t *testing.T) {
	local_host := NewLocalHost()
	ndh := local_host.ndh

	peer_target := NewNeighborDiscoveryTestPeer()
	world := NewWorld_Testimpl()

	peer_third := NewNeighborDiscoveryTestPeer()

	ndh.JoinAny("/", "noaddr", peer_target.GetHash(), "/w")

	ndh.Connected(peer_third)
	ndh.OnMEM(peer_third, world.GetUUID())

	ndh.Disconnected(peer_target.GetHash())

	time.Sleep(time.Second)
	for len(local_host.local_peer._log) > 0 {
		fmt.Println(<-local_host.local_peer._log)
	}
	fmt.Println("<<<join target" + peer_target.GetHash() + ">>>")
	for len(peer_target._log) > 0 {
		fmt.Println(<-peer_target._log)
	}
	fmt.Println("<<<third peer" + peer_third.GetHash() + ">>>")
	for len(peer_third._log) > 0 {
		fmt.Println(<-peer_third._log)
	}
}

func TestExpiredJoin1(t *testing.T) {
	local_host := NewLocalHost()
	ndh := local_host.ndh

	peer_target := NewNeighborDiscoveryTestPeer()
	world := NewWorld_Testimpl()

	peer_third := NewNeighborDiscoveryTestPeer()

	ndh.JoinAny("/", "noaddr", peer_target.GetHash(), "/w")

	ndh.Connected(peer_target)
	ndh.OnJOK(peer_target, "/w", world)

	ndh.CloseWorld("/")

	ndh.Connected(peer_third)
	ndh.OnMEM(peer_third, world.GetUUID())

	time.Sleep(time.Second)
	for len(local_host.local_peer._log) > 0 {
		fmt.Println(<-local_host.local_peer._log)
	}
	fmt.Println("<<<join target" + peer_target.GetHash() + ">>>")
	for len(peer_target._log) > 0 {
		fmt.Println(<-peer_target._log)
	}
	fmt.Println("<<<third peer" + peer_third.GetHash() + ">>>")
	for len(peer_third._log) > 0 {
		fmt.Println(<-peer_third._log)
	}
}

func TestExpiredJoin2(t *testing.T) {
	local_host := NewLocalHost()
	ndh := local_host.ndh

	peer_target := NewNeighborDiscoveryTestPeer()
	world := NewWorld_Testimpl()

	peer_third := NewNeighborDiscoveryTestPeer()

	ndh.JoinAny("/", "noaddr", peer_target.GetHash(), "/w")

	ndh.Connected(peer_target)
	ndh.OnJOK(peer_target, "/w", world)

	ndh.ChangeWorldPath("/", "/ss")
	ndh.CloseWorld("/ss")

	ndh.Connected(peer_third)
	ndh.OnMEM(peer_third, world.GetUUID())

	time.Sleep(time.Second)
	for len(local_host.local_peer._log) > 0 {
		fmt.Println(<-local_host.local_peer._log)
	}
	fmt.Println("<<<join target" + peer_target.GetHash() + ">>>")
	for len(peer_target._log) > 0 {
		fmt.Println(<-peer_target._log)
	}
	fmt.Println("<<<third peer" + peer_third.GetHash() + ">>>")
	for len(peer_third._log) > 0 {
		fmt.Println(<-peer_third._log)
	}
}

func TestAccept(t *testing.T) {
	local_host := NewLocalHost()
	ndh := local_host.ndh

	peer_target := NewNeighborDiscoveryTestPeer()
	world := NewWorld_Testimpl()

	ndh.OpenWorld("/home", world)

	ndh.Connected(peer_target)
	ndh.OnJN(peer_target, "/home")

	time.Sleep(time.Second)
	for len(local_host.local_peer._log) > 0 {
		fmt.Println(<-local_host.local_peer._log)
	}
	fmt.Println("<<<join target" + peer_target.GetHash() + ">>>")
	for len(peer_target._log) > 0 {
		fmt.Println(<-peer_target._log)
	}
}
