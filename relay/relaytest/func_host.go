package relaytest

import (
	"github.com/temporalio/tchannel-go"
	"github.com/temporalio/tchannel-go/relay"
)

// Ensure that the hostFunc implements tchannel.RelayHost and hostFuncPeer implements
// tchannel.RelayCall
var _ tchannel.RelayHost = (*hostFunc)(nil)
var _ tchannel.RelayCall = (*hostFuncPeer)(nil)

type hostFunc struct {
	ch    *tchannel.Channel
	stats *MockStats
	fn    func(relay.CallFrame, *relay.Conn) (string, error)
}

type hostFuncPeer struct {
	*MockCallStats

	peer      *tchannel.Peer
	respFrame relay.RespFrame
}

// HostFunc wraps a given function to implement tchannel.RelayHost.
func HostFunc(fn func(relay.CallFrame, *relay.Conn) (string, error)) tchannel.RelayHost {
	return &hostFunc{fn: fn}
}

func (hf *hostFunc) SetChannel(ch *tchannel.Channel) {
	hf.ch = ch
	hf.stats = NewMockStats()
}

func (hf *hostFunc) Start(cf relay.CallFrame, conn *relay.Conn) (tchannel.RelayCall, error) {
	var peer *tchannel.Peer

	peerHP, err := hf.fn(cf, conn)
	if peerHP != "" {
		peer = hf.ch.GetSubChannel(string(cf.Service())).Peers().GetOrAdd(peerHP)
	}

	// We still track stats if we failed to get a peer, so return the peer.
	return &hostFuncPeer{MockCallStats: hf.stats.Begin(cf), peer: peer}, err
}

func (hf *hostFunc) Stats() *MockStats {
	return hf.stats
}

func (p *hostFuncPeer) Destination() (*tchannel.Peer, bool) {
	return p.peer, p.peer != nil
}

func (p *hostFuncPeer) CallResponse(frame relay.RespFrame) {
	p.respFrame = frame
}
