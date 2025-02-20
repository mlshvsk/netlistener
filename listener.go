package netlistener

import (
	"net"
)

type (
	Listener struct {
		net.Listener
		config *bandwithConfig
	}
)

func NewListener(l net.Listener, globalLimit *int, perConnLimit *int) (*Listener, error) {
	return &Listener{
		Listener: l,
		config:   NewBandwithConfig(globalLimit, perConnLimit),
	}, nil
}

func (l *Listener) SetLimits(globalLimit int, perConnLimit int) {
	l.config.SetGlobalLimit(&globalLimit)
	l.config.SetPerConnLimit(&perConnLimit)
}

func (l *Listener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	return NewThrottledConnection(
		conn,
		NewConnectionBandwithConfig(l.config),
	), nil
}
