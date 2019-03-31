package stream

import (
	"sync"
	"nereuslch/stream/proto"
)

type Peer interface {
	Send(m interface{})

	AttachConn(conn *conn)

	Stop()

	Local() ID

	Remote() ID

	IsActivate() bool
}

type peer struct {
	localID  ID
	remoteID ID
	reader   *streamReader
	writer   *streamWriter
	msgChan  chan interface{}
	stopC    chan struct{}
	active   bool
	sync.RWMutex
}

func StartPeer(s *Server, remoteId ID, url string) Peer {
	p := &peer{
		localID:  s.serverId,
		remoteID: remoteId,
		msgChan:  make(chan interface{}),
		stopC:    make(chan struct{}),
	}

	p.writer = startStreamWriter(s.serverId, remoteId, p.activate, p.deactivate)
	p.reader = startStreamReader()
	return p
}

func (p *peer) Send(m interface{}) {
	p.writer.send(m.(proto.Message))
}

func (p *peer) AttachConn(c *conn) {
	if !p.writer.attach(c) {
		c.Close()
	}
}

func (p *peer) IsActivate() bool {
	p.RLock()
	defer p.RUnlock()
	return p.active
}

func (p *peer) Stop() {
	p.writer.stop()
	p.reader.stop()
}

func (p *peer) Local() ID {
	return p.localID
}

func (p *peer) Remote() ID {
	return p.remoteID
}

func (p *peer) activate() {
	p.Lock()
	defer p.Unlock()
	if !p.active {
		p.active = true
	}
}

func (p *peer) deactivate() {
	p.Lock()
	defer p.Unlock()
	if p.active {
		p.active = false
	}
}
