package stream

import (
	"net/http"
	"fmt"
)

type PeerGetter interface {
	Get(ID) Peer
}

type streamHandler struct {
	getter PeerGetter
}

func NewStreamHandler(pg PeerGetter) http.Handler { return &streamHandler{getter: pg} }

func (s *streamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.Header().Set("Allow_Method", "GET")
		http.Error(w, "Method not Allowed", http.StatusMethodNotAllowed)
		return
	}

	remoteID, err := IDFromString(r.Header.Get("Server-From"))
	if err != nil {
		http.Error(w, "Miss Server From ID", http.StatusForbidden)
		return
	}

	peer := s.getter.Get(remoteID)
	if peer == nil {
		http.Error(w, fmt.Sprintf("Invalid Server From ID %v", remoteID), http.StatusForbidden)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.(http.Flusher).Flush()

	notify := newCloseNotifier()
	peer.AttachConn(&conn{
		Writer:  w,
		Flusher: w.(http.Flusher),
		Closer:  notify,
	})
	<-notify.Notify()
}
