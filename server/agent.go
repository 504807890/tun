package server

import (
	"errors"
	"net"
	"time"

	"github.com/4396/tun/msg"
)

type Agent struct {
	net.Conn
	connc chan net.Conn
	donec chan interface{}
}

func NewAgent(conn net.Conn) (a *Agent) {
	a = &Agent{
		Conn:  conn,
		connc: make(chan net.Conn, 16),
		donec: make(chan interface{}),
	}
	return
}

func (a *Agent) Put(conn net.Conn) {
	a.connc <- conn
}

var (
	ErrAgentClosed  = errors.New("Agent closed")
	ErrNoEnoughConn = errors.New("No enough conn")
)

func (a *Agent) dial() (conn net.Conn, err error) {
	select {
	case conn = <-a.connc:
	case <-a.donec:
		err = ErrAgentClosed
	default:
		err = msg.Write(a.Conn, &msg.Dial{})
	}
	if conn != nil || err != nil {
		return
	}

	select {
	case conn = <-a.connc:
	case <-a.donec:
		err = ErrAgentClosed
	case <-time.After(time.Second):
		err = ErrNoEnoughConn
	}
	return
}

func (a *Agent) Dial() (conn net.Conn, err error) {
	if conn, err = a.dial(); err == nil {
		err = msg.Write(conn, &msg.StartWorkConn{})
	}
	return
}

func (a *Agent) Close() (err error) {
	close(a.donec)
	a.Conn.Close()
	for {
		select {
		case conn := <-a.connc:
			conn.Close()
		default:
			return
		}
	}
}