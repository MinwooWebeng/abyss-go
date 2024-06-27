package anet

import (
	"abyss/atype"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"math/big"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
)

type GoQuicNetCore struct {
	local_identity atype.AbyssIdentity
	ahmp_init_msg  []byte

	tlsConf  tls.Config
	quicConf quic.Config
	tr       quic.Transport
	ln       *quic.Listener

	listen_ctx    context.Context
	listen_cancel context.CancelFunc
	close_wg      sync.WaitGroup
}

func NewGoQuicNetCore(local_identity atype.AbyssIdentity) (*GoQuicNetCore, error) {
	result := new(GoQuicNetCore)
	result.local_identity = local_identity

	var sb strings.Builder
	sb.WriteString("AHMP/1.0 ID " + local_identity.Name + "\n")
	sb.WriteString("Content-Length: " + strconv.Itoa(len(local_identity.Publickey)) + "\n")
	sb.WriteString("\n")
	sb.Write(local_identity.Publickey)
	result.ahmp_init_msg = []byte(sb.String())

	listen_ctx, cancelfunc := context.WithCancel(context.Background())
	result.listen_ctx = listen_ctx
	result.listen_cancel = cancelfunc

	udpConn, err := net.ListenUDP("udp4", &net.UDPAddr{})
	if err != nil {
		return nil, err
	}

	ed25519_public_key, ed25519_private_key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	template := x509.Certificate{
		SerialNumber:          big.NewInt(0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, ed25519_public_key, ed25519_private_key)
	if err != nil {
		return nil, err
	}
	result.tlsConf = tls.Config{
		Certificates: []tls.Certificate{
			{
				Certificate: [][]byte{derBytes},
				PrivateKey:  ed25519_private_key,
			},
		},
		InsecureSkipVerify: true,
	}
	result.quicConf = quic.Config{
		MaxIdleTimeout:                time.Minute * 5,
		AllowConnectionWindowIncrease: func(conn quic.Connection, delta uint64) bool { return true },
		MaxIncomingStreams:            1000,
		MaxIncomingUniStreams:         1000,
		KeepAlivePeriod:               time.Minute,
		Allow0RTT:                     true,
		EnableDatagrams:               true,
	}
	result.tr = quic.Transport{
		Conn: udpConn,
	}
	ln, err := result.tr.Listen(&result.tlsConf, &result.quicConf)
	if err != nil {
		return nil, err
	}
	result.ln = ln

	return result, nil
}

func (n *GoQuicNetCore) Connect(abyss_address atype.AbyssAddress) (*Session, error) {
	n.close_wg.Add(1)
	defer n.close_wg.Done()
	var err error

	net_ipaddr, err := netip.ParseAddr(abyss_address.IP)
	if err != nil {
		return nil, err
	}

	dialctx, dialcancel := context.WithTimeout(n.listen_ctx, time.Second*3)
	connection, err := n.tr.Dial(
		dialctx,
		net.UDPAddrFromAddrPort(netip.AddrPortFrom(net_ipaddr, abyss_address.Port)),
		&n.tlsConf,
		&n.quicConf,
	)
	dialcancel()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			connection.CloseWithError(0x42, err.Error())
		}
	}()

	var new_peer *Session
	fin := make(chan error, 1)
	go func() {
		var err error
		defer func() {
			fin <- err
		}()

		ahmp_stream, err := connection.OpenStream()
		if err != nil {
			return
		}
		new_peer, err = NewSession(connection, ahmp_stream, n.ahmp_init_msg)
		if err != nil {
			return
		}
		if new_peer.identity.Hash != abyss_address.Pubkey_hash {
			err = errors.New("hash mismatch")
			return
		}
	}()

	//timeout
	select {
	case err = <-fin:
		if err != nil {
			return nil, err
		}
		return new_peer, nil
	case <-time.After(time.Second * 3):
		err = context.DeadlineExceeded
		return nil, err
	}
}
func (n *GoQuicNetCore) Accept() (*Session, error) {
	n.close_wg.Add(1)
	defer n.close_wg.Done()

	acceptctx, acceptcancel := context.WithTimeout(n.listen_ctx, time.Second*3)
	connection, err := n.ln.Accept(acceptctx)
	acceptcancel()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			connection.CloseWithError(0x42, "connection init failed")
		}
	}()

	var new_peer *Session
	fin := make(chan error, 1)
	go func() {
		var err error
		defer func() {
			fin <- err
		}()

		ahmp_stream, err := connection.AcceptStream(connection.Context())
		if err != nil {
			return
		}
		new_peer, err = NewSession(connection, ahmp_stream, n.ahmp_init_msg)
		if err != nil {
			return
		}
	}()

	//timeout
	select {
	case err = <-fin:
		if err != nil {
			return nil, err
		}
		return new_peer, nil
	case <-time.After(time.Second * 3):
		err = context.DeadlineExceeded
		return nil, err
	}
}
func (n *GoQuicNetCore) LocalIdentity() atype.AbyssIdentity {
	return n.local_identity
}
func (n *GoQuicNetCore) LocalAddr() atype.AbyssAddress {
	localaddr, _ := n.ln.Addr().(*net.UDPAddr)
	address, _ := atype.MakeAbyssAddress(n.local_identity.Hash, "127.0.0.1", localaddr.AddrPort().Port(), "")
	return address
}
func (n *GoQuicNetCore) Close() {
	n.listen_cancel()
	n.close_wg.Wait()
}
