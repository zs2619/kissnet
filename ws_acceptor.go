package kissnet

import (
	"fmt"
	"io"
	"net/http"
	"sync/atomic"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type WSAcceptor struct {
	Acceptor
	addr    string
	server  *http.Server
	running int32
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  8192,
	WriteBufferSize: 8192,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func NewWSAcceptor(port int, cb ClientCB) (*WSAcceptor, error) {
	addrStr := fmt.Sprintf(":%d", port)

	server := &http.Server{Addr: addrStr}
	return &WSAcceptor{addr: addrStr, server: server, Acceptor: Acceptor{ClientCB: cb}}, nil
}

func (this *WSAcceptor) Run() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "I Am WSAcceptor")
	})
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"error": err,
			}).Fatal("WSAcceptor::Run::Upgrade error")
			return
		}
		ws := newWSConnection(c, this)
		go ws.start()
	})
	this.server.Handler = mux

	go func() {
		atomic.StoreInt32(&this.running, 1)
		err := this.server.ListenAndServe()
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"error": err,
			}).Warn("WSAcceptor::Run::ListenAndServe error")
			atomic.StoreInt32(&this.running, 0)
			return
		}
	}()
	return nil
}

func (this *WSAcceptor) Close() error {
	this.server.Close()
	atomic.StoreInt32(&this.running, 0)
	return nil
}
func (this *WSAcceptor) IsRunning() bool {
	return atomic.LoadInt32(&this.running) > 0
}
