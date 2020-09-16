package sctp

import (
	"net"
	"sync"

	"github.com/pion/logging"
	"github.com/pion/udp"
	"github.com/pkg/errors"
)

// ListenAssociation creates a SCTP association listener.
func ListenAssociation(network string, laddr *net.UDPAddr) (*AssociationListener, error) {
	return (&ListenConfig{}).ListenAssociation(network, laddr)
}

// ListenAssociation creates a SCTP association listener.
func (lc *ListenConfig) ListenAssociation(network string, laddr *net.UDPAddr) (*AssociationListener, error) {
	parent, err := (&udp.ListenConfig{}).Listen(network, laddr)
	if err != nil {
		return nil, err
	}
	if lc.Config == nil {
		lc.Config = &Config{
			LoggerFactory: logging.NewDefaultLoggerFactory(),
		}
	}
	return &AssociationListener{
		config: *lc.Config,
		parent: parent,
	}, nil
}

// NewAssociationListener creates a SCTP association listener
// which accepts connections from an inner Listener.
func NewAssociationListener(inner net.Listener, config Config) (*AssociationListener, error) {
	return &AssociationListener{
		config: config,
		parent: inner,
	}, nil
}

// ListenConfig stores options for listening to an address.
// The net.Conn in the config is ignored.
type ListenConfig struct {
	Config *Config
}

// AssociationListener represents a SCTP association listener
type AssociationListener struct {
	config Config
	parent net.Listener
}

// Accept waits for and returns the next association to the listener.
// You have to either close or read on all connection that are created.
func (l *AssociationListener) Accept() (*Association, error) {
	c, err := l.parent.Accept()
	if err != nil {
		return nil, err
	}
	l.config.NetConn = c
	return Server(l.config)
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
// Already Accepted connections are not closed.
func (l *AssociationListener) Close() error {
	return l.parent.Close()
}

// Addr returns the listener's network address.
func (l *AssociationListener) Addr() net.Addr {
	return l.parent.Addr()
}

// Listen creates a basic SCTP stream listener.
// For more control check out the AssociationListener.
func Listen(network string, laddr *net.UDPAddr) (net.Listener, error) {
	return (&ListenConfig{}).Listen(network, laddr)
}

// Listen creates a basic SCTP stream listener.
// For more control check out the AssociationListener.
func (lc *ListenConfig) Listen(network string, laddr *net.UDPAddr) (net.Listener, error) {
	parent, err := ListenAssociation(network, laddr)
	if err != nil {
		return nil, err
	}

	if lc.Config == nil {
		lc.Config = &Config{
			LoggerFactory: logging.NewDefaultLoggerFactory(),
		}
	}

	l := &listener{
		config:    *lc.Config,
		parent:    parent,
		acceptCh:  make(chan *Stream),
		closeCh:   make(chan struct{}),
		serverMap: make(map[string]*Stream),
	}

	go l.run()

	return l, nil
}

func (l *listener) run() {
	defer close(l.closeCh)
	defer close(l.acceptCh)

	// TODO: Shutdown
	for {
		// TODO: Cleanup association when closing the last stream and/or listener.
		a, err := l.parent.Accept()
		if err != nil {
			// TODO: Error handling
			return
		}
		go l.acceptStreamLoop(a)
	}
}

func (l *listener) acceptStreamLoop(a *Association) {
	// TODO: Shutdown
	for {
		s, err := a.AcceptStream()
		if err != nil {
			// TODO: Error handling
			return
		}
		l.saveConn(s)
	}
}

func (l *listener) saveConn(s *Stream) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	_, ok := l.serverMap[s.String()]
	if !ok {
		select {
		case l.acceptCh <- s:
			l.serverMap[s.String()] = s
			break
		default:
			break
		}
	}
}

type listener struct {
	config Config
	parent *AssociationListener

	serverMap map[string]*Stream
	acceptCh  chan *Stream

	closeCh chan struct{}
	mutex   sync.RWMutex
}

var _ net.Listener = (*listener)(nil)

// Accept waits for and returns the next stream to the listener.
// You have to either close or read on all connection that are created.
func (l *listener) Accept() (net.Conn, error) {
	s, ok := <-l.acceptCh

	if !ok {
		return nil, errors.Errorf("listener closed")
	}
	return s, nil
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
// Already Accepted connections are not closed.
func (l *listener) Close() error {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	for k, v := range l.serverMap {
		v.Close()
		delete(l.serverMap, k)
	}
	return l.parent.Close()
}

// Addr returns the listener's network address.
func (l *listener) Addr() net.Addr {
	return l.parent.Addr()
}
