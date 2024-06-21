package and

type NeighborDiscoveryPeerState int

const (
	NC_NI NeighborDiscoveryPeerState = iota + 10
	AC_NI
	AC_MM
	AC_PM
	AC_JN
	CC_JN
	CC_MR
)

type INeighborDiscoveryPeerBase interface {
	SendJN(path string)
	SendJOK(path string, world INeighborDiscoveryWorldBase) //only 200 OK
	SendJDN(path string, status int, message string)        //this also includes redirection
	SendJNI(world INeighborDiscoveryWorldBase, member INeighborDiscoveryPeerBase)
	SendMEM(world INeighborDiscoveryWorldBase)
	SendSNB(world INeighborDiscoveryWorldBase, members []INeighborDiscoveryIdentityBase)
	SendCRR(world INeighborDiscoveryWorldBase, member INeighborDiscoveryIdentityBase)
	SendRST(world_uuid string)

	GetAddress() string
	GetIdentity() INeighborDiscoveryIdentityBase
	//GetJsonString() string
}
