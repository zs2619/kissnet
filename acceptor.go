package kissnet

import (
	"fmt"
)

type IAcceptor interface {
	Run() error
	Close() error
	IsRunning() bool
}

type ClientCB func(IConnection, []byte) error

type Acceptor struct {
	CMgr     ConnectionMgr
	ClientCB ClientCB
}

func AcceptorFactory(acceptorType string, port int, cb ClientCB) (IAcceptor, error) {
	if acceptorType == "ws" {
		return NewWSAcceptor(port, cb)
	} else if acceptorType == "tcp" {
		return NewTcpAcceptor(port, cb)
	} else {
		return nil, fmt.Errorf("config AcceptorType error")
	}
}
