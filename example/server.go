package main

import (
	"github.com/sirupsen/logrus"
	"github.com/zs2619/kissnet-go"
)

func testCB(kissnet.IConnection, []byte) error {
	return nil
}

func main() {
	event := kissnet.NewNetEvent()
	gAcceptor, err := kissnet.AcceptorFactory(
		"ws",
		2619,
		testCB,
	)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Fatal("AcceptorFactory error")
		return
	}

	gAcceptor.Run()
	event.EventLoop()
	gAcceptor.Close()
}
