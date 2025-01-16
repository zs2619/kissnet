package kissnet

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"sync"
	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const MsgHeaderMaxSize = 2

type Connection struct {
	id           int64
	conn         *net.TCPConn
	exitSync     sync.WaitGroup
	cb           *CallBack
	isClose      int32
	sendCh       chan *bytes.Buffer
	lastPingTime int64
}

func NewConnection(conn *net.TCPConn, cb *CallBack) IConnection {
	c := &Connection{
		conn:    conn,
		isClose: 0,
		cb:      cb,
		sendCh:  make(chan *bytes.Buffer, 1024),
	}
	return c
}

func (c *Connection) getID() int64 {
	return c.id
}
func (c *Connection) setID(id int64) {
	c.id = id
}

func (c *Connection) IsClose() bool { return atomic.LoadInt32(&c.isClose) > 0 }

func (c *Connection) Close() {
	c.cb.ConnectionCB(c, nil)
	c.SendMsg(nil)

	c.exitSync.Wait()
	atomic.StoreInt32(&c.isClose, 1)

	close(c.sendCh)
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	logrus.WithFields(logrus.Fields{"id:": c.id}).Info("TcpConnection  Close")
}

func (c *Connection) start() {
	c.conn.SetNoDelay(true)
	c.conn.SetKeepAlive(true)

	//同步退出 goroutine
	c.exitSync.Add(2)

	//开启读写 goroutine
	go c.recvMsgLoop()
	go c.sendMsgLoop()
}

func (c *Connection) SendMsg(msg *bytes.Buffer) error {
	if c.IsClose() {
		//关闭不能发送消息
		return nil
	}
	//推入发送循环
	c.sendCh <- msg
	return nil
}

func (c *Connection) sendMsgLoop() {
	defer func() {
		if err := recover(); err != nil {
			c.exitSync.Done()
			if e, ok := err.(error); ok {
				logrus.WithError(errors.WithStack(e)).Error("sendMsgLoop error")
			}
		}
	}()
	for msg := range c.sendCh {
		if msg == nil || c.conn == nil {
			break
		}

		msgLen := uint16(msg.Len())
		buf := make([]byte, MsgHeaderMaxSize)
		binary.LittleEndian.PutUint16(buf, msgLen)
		_, err := c.conn.Write(buf)
		if err != nil {
			break
		}
		_, err = c.conn.Write(msg.Bytes())
		if err != nil {
			break
		}
	}
	//关闭socket 从读操作退出
	c.exitSync.Done()
}

func (c *Connection) recvMsgLoop() {
	defer func() {
		if err := recover(); err != nil {
			c.exitSync.Done()
			// 退出处理
			c.Close()
			//打印堆栈
			if e, ok := err.(error); ok {
				logrus.WithError(errors.WithStack(e)).Error("recvMsgLoop error")
			}
		}
	}()
	var err error
	var msgLen int
	msgHeader := make([]byte, MsgHeaderMaxSize)
	for {
		_, err = io.ReadFull(c.conn, msgHeader)
		if err != nil {
			break
		}
		msgLen = int(binary.LittleEndian.Uint16(msgHeader))
		if msgLen <= 0 || msgLen > 65535 {
			break
		}

		msgBody := make([]byte, msgLen)
		_, err = io.ReadFull(c.conn, msgBody)
		if err != nil {
			break
		}
		c.cb.ConnectionCB(c, msgBody)
	}
}
