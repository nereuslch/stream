package stream

import (
	"nereuslch/stream/proto"
	"strings"
)

var (
	LinkHeartbeatMessage = proto.Message{Type: proto.MessageType_MsgHeartbeat}
)

func isLinkHeartbeatMessage(message *proto.Message) bool {
	return message.Type == proto.MessageType_MsgHeartbeat
}

func isClosedConnError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "closed")
}

type closeNotifier struct {
	done chan struct{}
}

func newCloseNotifier() *closeNotifier {
	return &closeNotifier{
		done: make(chan struct{}),
	}
}

func (c *closeNotifier) Close() error {
	close(c.done)
	return nil
}

func (c *closeNotifier) Notify() <-chan struct{} {
	return n.done
}
