package stream

import (
	"io"
	"net/http"
	"time"
	"nereuslch/stream/proto"
	"sync"
)

type conn struct {
	io.Writer
	io.Closer
	http.Flusher
}

type streamWriter struct {
	localId      ID
	remoteId     ID
	connC        chan *conn
	msgC         chan proto.Message
	closer       io.Closer
	activeFunc   func()
	deactiveFunc func()
	stopc        chan struct{}
	stoped       chan struct{}
	working      bool
	lock         sync.RWMutex
}

func startStreamWriter(lid, rid ID, af, daf func()) *streamWriter {
	w := &streamWriter{}
	go w.run()
	return w
}

const (
	ConnReadTimeOut = 6 * time.Second
	SizeStreamBuff  = 4096
)

func (s *streamWriter) run() {
	var (
		heartbeatc <-chan time.Time
		msgc       chan proto.Message
		flusher    http.Flusher
		enc        Encoder
		dataSize   int
		batched    int
	)

	ticker := time.NewTicker(ConnReadTimeOut / 2)
	defer func() {
		ticker.Stop()
		close(s.stoped)
	}()

	for {
		select {
		case <-heartbeatc:
			err := enc.Encode(&LinkHeartbeatMessage, LinkHeartbeatMessage.Size())
			if err == nil {
				flusher.Flush()
				dataSize = 0
				batched = 0
				break
			}

			s.deactiveFunc()
			s.close()
			heartbeatc, msgc = nil, nil
		case m := <-msgc:
			err := enc.Encode(m, m.Size())
			if err == nil {
				dataSize += m.Size()
				if len(msgc) == 0 || batched > SizeStreamBuff/2 {
					flusher.Flush()
					batched = 0
					dataSize = 0
				} else {
					batched ++
				}
				break
			}
			s.deactiveFunc()
			s.close()
			heartbeatc, msgc = nil, nil
		case conn := <-s.connC:
			s.lock.Lock()
			closed := s.closeUnlocked()
			enc = NewMsgEncoder(conn.Writer)
			flusher = conn.Flusher
			s.closer = conn.Closer
			dataSize = 0
			s.activeFunc()
			s.working = true
			s.lock.Unlock()

			if closed {
				// ADD LOG
			}

			heartbeatc, msgc = ticker.C, s.msgC
		case <-s.stopc:
			s.close()
			return
		}
	}
}

func (s *streamWriter) stop() {
	close(s.stopc)
	<-s.stoped
}

func (s *streamWriter) closeUnlocked() bool {
	if !s.working {
		return false
	}
	s.closer.Close()
	s.msgC = make(chan proto.Message, SizeStreamBuff)
	s.working = false
	return true
}

func (s *streamWriter) close() bool {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.closeUnlocked()
}

func (s *streamWriter) send(m proto.Message) {
	select {
	case s.msgC <- m:
	default:
	}
}

func (s *streamWriter) attach(c *conn) bool {
	select {
	case s.connC <- c:
		return true
	case <-s.stopc:
		return false
	}
}
