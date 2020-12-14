package kissnet

import (
	"bytes"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// websocket会话
type WsConnection struct {
	id           int64 //会话id
	conn         *websocket.Conn
	isClose      int32
	sendCh       chan *bytes.Buffer
	acceptor     *WSAcceptor
	exitSync     sync.WaitGroup
	lastPingTime int64
}

const (
	recvWait = 120 * time.Second
)

func (this *WsConnection) getID() int64 {
	return this.id
}
func (this *WsConnection) setID(id int64) {
	this.id = id
}

func newWSConnection(conn *websocket.Conn, acceptor *WSAcceptor) IConnection {
	c := &WsConnection{
		conn:     conn,
		isClose:  0,
		sendCh:   make(chan *bytes.Buffer, 2048),
		acceptor: acceptor,
	}
	c.conn.SetPingHandler(
		func(message string) error {
			c.conn.SetReadDeadline(time.Now().Add(recvWait))
			err := c.conn.WriteControl(websocket.PongMessage, []byte(message), time.Now().Add(time.Second))
			if err == websocket.ErrCloseSent {
				return nil
			} else if e, ok := err.(net.Error); ok && e.Temporary() {
				return nil
			}
			return err
		})
	return c
}

func (this *WsConnection) IsClose() bool { return atomic.LoadInt32(&this.isClose) > 0 }

func (this *WsConnection) Close() {
	this.acceptor.ClientCB(this, nil)

	this.SendMsg(nil)

	this.exitSync.Wait()

	atomic.StoreInt32(&this.isClose, 1)
	close(this.sendCh)
	if this.conn != nil {
		this.conn.Close()
		this.conn = nil
	}
	logrus.WithFields(logrus.Fields{"id:": this.id}).Info("WsConnection  Close")
}

func (this *WsConnection) SendMsg(msg *bytes.Buffer) error {
	if this.IsClose() {
		//关闭不能发送消息
		return nil
	}
	//推入发送循环
	this.sendCh <- msg
	return nil
}

func (this *WsConnection) GetId() int64 {
	return this.id
}

func (this *WsConnection) SetID(id int64) {
	this.id = id
}

func (this *WsConnection) start() {
	this.exitSync.Add(2)
	//开启读写 goroutine
	go this.recvMsgLoop()
	go this.sendMsgLoop()

}

func (this *WsConnection) sendMsgLoop() {
	for msg := range this.sendCh {
		if msg == nil || this.conn == nil {
			break
		}
		err := this.conn.WriteMessage(websocket.BinaryMessage, msg.Bytes())
		if err != nil {
			logrus.Error(err.Error())
		}
	}
	this.exitSync.Done()
}

func (this *WsConnection) recvMsgLoop() {
	for {
		if this.conn == nil {
			logrus.Error("this.conn")
			break
		}

		this.conn.SetReadDeadline(time.Now().Add(recvWait))
		mt, msg, err := this.conn.ReadMessage()
		if err != nil {
			logrus.Error(err.Error())
			break
		}
		if mt == websocket.BinaryMessage {
			err := this.acceptor.ClientCB(this, msg)
			if err != nil {
				logrus.Error(err)
				break
			}
		} else {
			err := fmt.Errorf("message type error")
			logrus.Error(err.Error())
			break
		}
	}
	this.exitSync.Done()
	// 退出处理
	this.Close()
}
