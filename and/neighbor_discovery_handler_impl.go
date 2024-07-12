package and

import (
	"errors"
	"time"

	distuv_rand "golang.org/x/exp/rand"
	"gonum.org/v1/gonum/stat/distuv"
)

type NeighborDiscoverySession struct {
	world INeighborDiscoveryWorldBase

	//key: identity hash
	members        map[string]INeighborDiscoveryPeerBase //AC_MM, AC_PM(candidate sessions)
	CC_MR          map[string]INeighborDiscoveryPeerBase
	snb_targets    map[string]int //decrement on SNB
	is_snb_planned bool
}

type CandidateSession struct {
	members map[string]INeighborDiscoveryPeerBase
}

func NewCandidateSession() *CandidateSession {
	result := new(CandidateSession)
	result.members = make(map[string]INeighborDiscoveryPeerBase)
	return result
}

func NewNeighborDiscoverySession() *NeighborDiscoverySession {
	result := new(NeighborDiscoverySession)
	result.members = make(map[string]INeighborDiscoveryPeerBase)
	result.CC_MR = make(map[string]INeighborDiscoveryPeerBase)
	result.snb_targets = make(map[string]int)
	result.is_snb_planned = false
	return result
}

type NeighborDiscoveryHandler struct {
	event_listener   chan<- NeighborDiscoveryEvent
	error_listener   chan<- error
	connect_callback func(address any)
	snb_timer        func(time.Duration, string)
	snb_randsrc      distuv_rand.Source

	local_hash string

	peers    map[string]INeighborDiscoveryPeerBase  //identity hash > peer - all connected peers
	worlds   map[string]INeighborDiscoveryWorldBase //localpath > world - same world for different localpath is not allowed.
	sessions map[string]*NeighborDiscoverySession   //world UUID > session - only one session for same world UUID.

	candidate_sessions map[string]*CandidateSession //candidate join target sessions. non empty only if there is ongoing join process (previously, AC_PM)

	join_targets     map[string]map[string]string //identity hash > set of join paths > local path - AC_JN, CC_JN //TODO: implement expiration timer
	join_local_paths map[string]bool              //occupied local paths
}

func NewNeighborDiscoveryHandler(local_hash string) *NeighborDiscoveryHandler {
	result := new(NeighborDiscoveryHandler)
	result.snb_randsrc = distuv_rand.NewSource(uint64(time.Now().UTC().UnixNano()))
	result.local_hash = local_hash
	result.peers = make(map[string]INeighborDiscoveryPeerBase)
	result.worlds = make(map[string]INeighborDiscoveryWorldBase)
	result.sessions = make(map[string]*NeighborDiscoverySession)
	result.candidate_sessions = make(map[string]*CandidateSession)
	result.join_targets = make(map[string]map[string]string)
	result.join_local_paths = make(map[string]bool)
	return result
}

func (h *NeighborDiscoveryHandler) ReserveEventListener(listener chan<- NeighborDiscoveryEvent) {
	h.event_listener = listener
}
func (h *NeighborDiscoveryHandler) ReserveErrorListener(listener chan<- error) {
	h.error_listener = listener
}
func (h *NeighborDiscoveryHandler) ReserveConnectCallback(connect_callback func(address any)) {
	h.connect_callback = connect_callback
}
func (h *NeighborDiscoveryHandler) ReserveSNBTimer(snb_timer func(time.Duration, string)) {
	h.snb_timer = snb_timer
}
func (h *NeighborDiscoveryHandler) SetSNBTimer(session *NeighborDiscoverySession) {
	if !session.is_snb_planned {
		h.snb_timer(time.Millisecond*time.Duration(distuv.Weibull{K: 0.72, Lambda: 800 * float64(len(session.members)+1), Src: h.snb_randsrc}.Rand()), session.world.GetUUID())
	}
}

func (h *NeighborDiscoveryHandler) IsLocalPathOccupied(localpath string) bool {
	_, ok := h.worlds[localpath]
	if ok {
		return true
	}

	_, ok = h.join_local_paths[localpath]
	return ok
}

func (h *NeighborDiscoveryHandler) OpenWorld(localpath string, world INeighborDiscoveryWorldBase) bool {
	if h.IsLocalPathOccupied(localpath) {
		h.error_listener <- errors.New("local path collision in OpenWorld: " + localpath)
		return false
	}

	//world UUID collision; practically, never gonna happen. each newly opened worlds must have different UUID
	_, ok := h.sessions[world.GetUUID()]
	if ok {
		return false
	}
	_, ok = h.candidate_sessions[world.GetUUID()]
	if ok {
		return false
	}

	h.worlds[localpath] = world
	session := NewNeighborDiscoverySession()
	session.world = world
	h.sessions[world.GetUUID()] = session
	return true
}
func (h *NeighborDiscoveryHandler) CloseWorld(localpath string) {
	world, ok := h.worlds[localpath]
	if !ok {
		return
	}

	session, ok := h.sessions[world.GetUUID()]
	if !ok {
		h.error_listener <- errors.New("missing session in CloseWorld")
		return
	}
	for _, member := range session.members {
		member.SendRST(world.GetUUID())
	}

	delete(h.sessions, world.GetUUID())
	delete(h.worlds, localpath)
}

func (h *NeighborDiscoveryHandler) _OpenWorldOrLoadCandidateSession(localpath string, world INeighborDiscoveryWorldBase) (bool, *NeighborDiscoverySession) {
	_, ok := h.worlds[localpath]
	if ok {
		h.error_listener <- errors.New("local path collision in _OpenWorldOrLoadCandidateSession: " + localpath)
		return false, nil
	}

	h.worlds[localpath] = world

	var session = NewNeighborDiscoverySession()
	candidate_session, ok := h.candidate_sessions[world.GetUUID()]
	if ok {
		delete(h.candidate_sessions, world.GetUUID())
		session.members = candidate_session.members
	}
	session.world = world
	h.sessions[world.GetUUID()] = session
	return true, session
}

func (h *NeighborDiscoveryHandler) ChangeWorldPath(prev_localpath string, new_localpath string) bool {
	_, ok := h.worlds[new_localpath]
	if ok {
		return false
	}

	result, ok := h.worlds[prev_localpath]
	if !ok {
		return false
	}
	h.worlds[new_localpath] = result
	delete(h.worlds, prev_localpath)
	return true
}
func (h *NeighborDiscoveryHandler) GetWorld(localpath string) (INeighborDiscoveryWorldBase, bool) {
	result, ok := h.worlds[localpath]
	return result, ok
}

func (h *NeighborDiscoveryHandler) Connected(peer INeighborDiscoveryPeerBase) {
	//add in peers
	peer_id_hash := peer.GetHash()
	_, ok := h.peers[peer_id_hash]
	if ok {
		//error: duplicate connection
		h.error_listener <- errors.New("duplicate connection: " + peer.GetHash())
		return
	}
	if peer_id_hash == h.local_hash {
		h.error_listener <- errors.New("self connection")
		return
	}
	h.peers[peer_id_hash] = peer

	//look for all sessions, CC_MR to member.
	for _, session := range h.sessions {
		_, ok := session.CC_MR[peer_id_hash]
		if ok {
			delete(session.CC_MR, peer_id_hash)
			peer.SendMEM(session.world)
			session.members[peer_id_hash] = peer
			session.snb_targets[peer_id_hash] = 3
			h.SetSNBTimer(session)
			h.event_listener <- NeighborDiscoveryEvent{PeerJoin, "", peer.GetHash(), peer, "", session.world, 0, ""}
		}
	}

	//look for join targets -> send JN
	paths, ok := h.join_targets[peer_id_hash]
	if ok {
		for path := range paths {
			peer.SendJN(path)
		}
	}
}
func (h *NeighborDiscoveryHandler) Disconnected(peer_hash string) {
	//remove from peers
	delete(h.peers, peer_hash)

	//look for all sessions, remove
	for _, session := range h.sessions {
		peer, ok := session.members[peer_hash]
		if ok {
			delete(session.members, peer_hash)
			h.event_listener <- NeighborDiscoveryEvent{PeerLeave, "", peer_hash, peer, "", session.world, 0, ""}
		}

		delete(session.CC_MR, peer_hash)
		delete(session.snb_targets, peer_hash)
	}

	//candidate sessions, remove silently.
	for _, session := range h.candidate_sessions {
		_, ok := session.members[peer_hash]
		if ok {
			delete(session.members, peer_hash)
		}
	}

	//delete from join targets
	join_target, ok := h.join_targets[peer_hash]
	if ok {
		delete(h.join_targets, peer_hash)
		for path, localpath := range join_target {
			h.event_listener <- NeighborDiscoveryEvent{JoinExpired, localpath, peer_hash, nil, path, nil, 0, ""}
			delete(h.join_local_paths, localpath)
		}

		//all join processes terminated
		if len(h.join_local_paths) == 0 && len(h.candidate_sessions) != 0 {
			for candidate_uuid, candidate_session := range h.candidate_sessions {
				for _, candidate_member := range candidate_session.members {
					candidate_member.SendRST(candidate_uuid)
				}
			}
			h.candidate_sessions = make(map[string]*CandidateSession)
		}
	}
}

// path collision not checked
func (h *NeighborDiscoveryHandler) _AppendJoinInfo(localpath string, peer_hash string, path string) {
	join_paths, ok := h.join_targets[peer_hash]
	if !ok {
		//there is no ongoing join process
		join_paths = make(map[string]string)
		h.join_targets[peer_hash] = join_paths
	}

	_, ok = join_paths[path]
	if ok {
		h.event_listener <- NeighborDiscoveryEvent{JoinExpired, localpath, peer_hash, nil, path, nil, 0, ""}
		h.error_listener <- errors.New("duplicate join call: " + peer_hash + path)
		return
	}
	join_paths[path] = localpath
	h.join_local_paths[localpath] = true
}
func (h *NeighborDiscoveryHandler) JoinConnected(localpath string, peer INeighborDiscoveryPeerBase, path string) {
	if h.IsLocalPathOccupied(localpath) {
		h.event_listener <- NeighborDiscoveryEvent{JoinExpired, localpath, peer.GetHash(), peer, path, nil, 0, ""}
		h.error_listener <- errors.New("local path collision in JoinAny: " + localpath)
		return
	}

	_, ok := h.peers[peer.GetHash()]
	if !ok {
		h.error_listener <- errors.New("tried to join hanging peer")
		return
	}
	peer.SendJN(path)
	h._AppendJoinInfo(localpath, peer.GetHash(), path)
}
func (h *NeighborDiscoveryHandler) JoinAny(localpath string, address any, peer_hash string, path string) {
	if h.IsLocalPathOccupied(localpath) {
		h.event_listener <- NeighborDiscoveryEvent{JoinExpired, localpath, peer_hash, nil, path, nil, 0, ""}
		h.error_listener <- errors.New("local path collision in JoinAny: " + localpath)
		return
	}

	peer, ok := h.peers[peer_hash]
	if ok {
		peer.SendJN(path)
		h._AppendJoinInfo(localpath, peer_hash, path)
		return
	}

	h.connect_callback(address)
	h._AppendJoinInfo(localpath, peer_hash, path)
}

func (h *NeighborDiscoveryHandler) OnJN(peer INeighborDiscoveryPeerBase, path string) {
	world, ok := h.worlds[path]
	if !ok {
		//world not found.
		peer.SendJDN(path, 404, "Not Found")
		return
	}

	session, ok := h.sessions[world.GetUUID()]
	if !ok {
		peer.SendJDN(path, 500, "Internal Server Error")
		return
	}
	//duplicate join check
	_, ok = session.members[peer.GetHash()]
	if ok {
		peer.SendJDN(path, 409, "Conflict")
		return
	}

	for _, member := range session.members {
		member.SendJNI(world, peer)
	}

	session.members[peer.GetHash()] = peer
	peer.SendJOK(path, world)
	h.event_listener <- NeighborDiscoveryEvent{PeerJoin, "", peer.GetHash(), peer, "", session.world, 0, ""}
}
func (h *NeighborDiscoveryHandler) OnJOK(peer INeighborDiscoveryPeerBase, path string, world INeighborDiscoveryWorldBase) {
	//check for ongoing join processes
	join_paths, ok := h.join_targets[peer.GetHash()]
	if !ok {
		peer.SendRST(world.GetUUID())
		return
	}

	localpath, ok := join_paths[path]
	if !ok {
		peer.SendRST(world.GetUUID())
		return
	}

	ok, session := h._OpenWorldOrLoadCandidateSession(localpath, world)
	if !ok {
		peer.SendRST(world.GetUUID())
		return
	}

	h.event_listener <- NeighborDiscoveryEvent{JoinSuccess, localpath, peer.GetHash(), peer, path, world, 200, "OK"}
	session.members[peer.GetHash()] = peer
	for _, peer := range session.members {
		h.event_listener <- NeighborDiscoveryEvent{PeerJoin, "", peer.GetHash(), peer, "", world, 0, ""}
	}

	delete(join_paths, path)
	if len(join_paths) == 0 {
		delete(h.join_targets, peer.GetHash())
	}
	delete(h.join_local_paths, localpath)
	if len(h.join_local_paths) == 0 && len(h.candidate_sessions) != 0 {
		for candidate_uuid, candidate_session := range h.candidate_sessions {
			for _, candidate_member := range candidate_session.members {
				candidate_member.SendRST(candidate_uuid)
			}
		}
		h.candidate_sessions = make(map[string]*CandidateSession)
	}
}
func (h *NeighborDiscoveryHandler) OnJDN(peer INeighborDiscoveryPeerBase, path string, status int, message string) {
	//check for ongoing join processes
	join_paths, ok := h.join_targets[peer.GetHash()]
	if !ok {
		return
	}

	localpath, ok := join_paths[path]
	if !ok {
		return
	}

	h.event_listener <- NeighborDiscoveryEvent{JoinDenied, localpath, peer.GetHash(), peer, path, nil, status, message}

	delete(join_paths, path)
	if len(join_paths) == 0 {
		delete(h.join_targets, peer.GetHash())
	}
	delete(h.join_local_paths, localpath)
	if len(h.join_local_paths) == 0 && len(h.candidate_sessions) != 0 {
		for candidate_uuid, candidate_session := range h.candidate_sessions {
			for _, candidate_member := range candidate_session.members {
				candidate_member.SendRST(candidate_uuid)
			}
		}
		h.candidate_sessions = make(map[string]*CandidateSession)
	}
}

func (h *NeighborDiscoveryHandler) ValidateSessionMember(peer INeighborDiscoveryPeerBase, world_uuid string) (*NeighborDiscoverySession, *CandidateSession) {
	//check if session exists
	session, ok := h.sessions[world_uuid]
	if !ok { //session not exist
		//check if target session is candidate session
		candidate_session, ok := h.candidate_sessions[world_uuid]
		if ok {
			_, ok = candidate_session.members[peer.GetHash()]
			if ok { //premature JNI. there is little to no gain on handling this, though this may be a valid JNI.
				return nil, candidate_session
			}
		}

		//no matching session found
		return nil, nil
	}

	//check if the sender is member of target session.
	_, ok = session.members[peer.GetHash()]
	if !ok {
		return nil, nil
	}
	return session, nil
}
func (h *NeighborDiscoveryHandler) OnJNI(peer INeighborDiscoveryPeerBase, world_uuid string, address any, joiner_hash string) {
	session, candidate := h.ValidateSessionMember(peer, world_uuid)
	if session == nil && candidate == nil {
		peer.SendRST(world_uuid)
		return
	}
	if candidate != nil {
		return
	}

	//check if joiner is connected
	joiner, ok := h.peers[joiner_hash]
	if ok {
		//check if joiner is already member.
		_, ok = session.members[joiner_hash]
		if ok { //already member
			return //ignore duplicate JNI
		}

		//already connected, not a member
		joiner.SendMEM(session.world)
		session.members[joiner_hash] = joiner
		session.snb_targets[joiner_hash] = 3
		h.SetSNBTimer(session)
		return
	}

	//check if joiner is CC_MR
	_, ok = session.CC_MR[joiner_hash]
	if ok {
		return //duplicate JNI. all peers in CC_MR is already dialing
	}

	//not connected, not CC_MR
	h.connect_callback(address)
	session.CC_MR[joiner_hash] = joiner
}
func (h *NeighborDiscoveryHandler) OnMEM(peer INeighborDiscoveryPeerBase, world_uuid string) {
	session, ok := h.sessions[world_uuid]
	if !ok {
		//check if there is ongoing join process
		if len(h.join_local_paths) == 0 {
			peer.SendRST(world_uuid)
			return
		}

		candidate, ok := h.candidate_sessions[world_uuid]
		if !ok {
			candidate = NewCandidateSession()
			h.candidate_sessions[world_uuid] = candidate
		}
		candidate.members[peer.GetHash()] = peer
		return
	}
	_, ok = session.members[peer.GetHash()]
	if ok {
		return
	}

	session.members[peer.GetHash()] = peer
	h.event_listener <- NeighborDiscoveryEvent{PeerJoin, "", peer.GetHash(), peer, "", session.world, 0, ""}
}
func (h *NeighborDiscoveryHandler) OnSNB(peer INeighborDiscoveryPeerBase, world_uuid string, members_hash []string) {
	session, candidate := h.ValidateSessionMember(peer, world_uuid)
	if session == nil && candidate == nil {
		peer.SendRST(world_uuid)
		return
	}
	if candidate != nil {
		return
	}

	//now, peer is valid, handle SNB.
	for _, member_hash := range members_hash {
		cnt, ok := session.snb_targets[member_hash]
		if ok {
			if cnt == 1 {
				delete(session.snb_targets, member_hash)
			} else {
				session.snb_targets[member_hash] = cnt - 1
			}
			continue
		}

		//the member_id was not in snb_targets. check if it is also missing in members.
		_, ok = session.members[member_hash]
		if !ok {
			peer.SendCRR(session.world, member_hash)
		}
	}
}
func (h *NeighborDiscoveryHandler) OnCRR(peer INeighborDiscoveryPeerBase, world_uuid string, missing_member_hash string) {
	session, candidate := h.ValidateSessionMember(peer, world_uuid)
	if session == nil && candidate == nil {
		peer.SendRST(world_uuid)
		return
	}
	if candidate != nil {
		return
	}

	member, ok := session.members[missing_member_hash]
	if !ok {
		return
	}

	peer.SendJNI(session.world, member)
}
func (h *NeighborDiscoveryHandler) OnRST(peer INeighborDiscoveryPeerBase, world_uuid string) {
	session, candidate := h.ValidateSessionMember(peer, world_uuid)
	if session != nil {
		delete(session.members, peer.GetHash())
		delete(session.snb_targets, peer.GetHash())
	}
	if candidate != nil {
		delete(candidate.members, peer.GetHash())
	}
}
func (h *NeighborDiscoveryHandler) OnWorldErr(peer INeighborDiscoveryPeerBase, world_uuid string) {
	peer.SendRST(world_uuid)
	h.OnRST(peer, world_uuid)
}
func (h *NeighborDiscoveryHandler) OnSNBTimeout(world_uuid string) {
	session, ok := h.sessions[world_uuid]
	if !ok {
		return
	}
	defer func() { session.is_snb_planned = false }()

	if len(session.snb_targets) == 0 {
		return
	}

	snb_targets := make([]string, 0, len(session.snb_targets))
	for k := range session.snb_targets {
		snb_targets = append(snb_targets, k)
	}
	for _, member := range session.members {
		member.SendSNB(session.world, snb_targets)
	}

	session.snb_targets = make(map[string]int)
}
