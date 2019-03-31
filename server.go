package stream

import (
	"net/http"
	"sync"
)

type Server struct {
	streamRT http.RoundTripper
	peers    map[ID]Peer
	serverId ID
	option   *ServerOption
	lock     sync.RWMutex
}



type ServerOption struct {
	LocalOpt *pconf
	PeersOpt []*pconf
}

type pconf struct {
	IDstring string
	Url      string
}

func (p *pconf) ID() (id ID, err error) {
	id, err = IDFromString(p.IDstring)
	return
}
