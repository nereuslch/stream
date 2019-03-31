package stream

import (
	"io"
	"context"
	"errors"
	"net/http"
	"nereuslch/stream/proto"
	"golang.org/x/time/rate"
	"sync"
	"fmt"
)

type streamReader struct {
	server       *Server
	url          string
	id           ID
	ctx          context.Context
	cancel       context.CancelFunc
	closer       io.Closer
	limiter      *rate.Limiter
	activeFunc   func()
	deactiveFunc func()
	lock         sync.RWMutex
	messageChan  chan<- proto.Message
	done         chan struct{}
	closeOnce    sync.Once
}

func startStreamReader() *streamReader {
	r := &streamReader{}
	r.ctx, r.cancel = context.WithCancel(context.Background())
	return r
}

func (s *streamReader) dial() (io.ReadCloser, error) {
	var (
		req  *http.Request
		resp *http.Response
		err  error
	)

	select {
	case <-s.ctx.Done():
		return nil, errors.New("stream reader is close")
	default:
		req, err = http.NewRequest("GET", s.url, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("server-from", s.server.serverId.String())
		req.Header.Set("server-to", s.id.String())
		req.WithContext(s.ctx)
	}

	resp, err = s.server.streamRT.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return resp.Body, nil
	default:
		return nil, fmt.Errorf("unhandled http status %d", resp.StatusCode)
	}
}

func (s *streamReader) decodeLoop(rc io.ReadCloser) error {
	dec := NewMsgDecoder(rc)
	select {
	case s.ctx.Done():
		if err := rc.Close(); err != nil {
			return err
		}
		return io.EOF
	default:
		s.closer = rc
	}

	defer func() {
		s.close()
	}()

	for {
		var m proto.Message
		err := dec.Decode(&m)
		if err != nil {
			return err
		}

		if isLinkHeartbeatMessage(&m) {
			continue
		}

		select {
		case s.messageChan <- m:
		default:
			// TODO add log
			// dropped the message when recevie buffer is full
		}
	}
}

func (s *streamReader) close() {
	s.closeOnce.Do(func() {
		s.lock.Lock()
		defer s.lock.Unlock()
		if s.closer != nil {
			s.closer.Close()
		}
		s.closer = nil
	})
}

func (s *streamReader) run() {
	defer close(s.done)
	for {
		rc, err := s.dial()
		if err != nil {
			// TODO add log
		} else {
			s.activeFunc()
			err := s.decodeLoop(rc)

			switch {
			case err == io.EOF:
			case isClosedConnError(err):
			default:
				s.deactiveFunc()
			}
		}

		err = s.limiter.Wait(s.ctx)
		if s.ctx.Err() != nil {
			s.close()
			return
		}
	}
}

func (s *streamReader) stop() {
	s.cancel()
	s.close()
	<-s.done
}
