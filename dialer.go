package sctp

import (
	"net"

	"github.com/pion/logging"
)

// Dial connects to the given network address and establishes a
// SCTP stream on top. For more control use DialAssociation.
func Dial(network string, raddr *net.UDPAddr, streamIdentifier uint16) (*Stream, error) {
	return (&Dialer{}).Dial(network, raddr, streamIdentifier)
}

// func new Dial
func NewDialer(unordered bool, relType byte, relVal uint32) *Dialer {
	cfg := &Reliability{unordered: unordered, reliabilityType: relType, reliabilityValue: relVal}
	d := &Dialer{ReliabilityConfig: cfg}
	return d
}

// A Dialer contains options for connecting to an address.
//
// The zero value for each field is equivalent to dialing without that option.
// Dialing with the zero value of Dialer is therefore equivalent
// to just calling the Dial function.
//
// The net.Conn in the config is ignored.
type Dialer struct {
	// PayloadType determines the PayloadProtocolIdentifier used
	PayloadType PayloadProtocolIdentifier

	// Config holds common config
	Config *Config

	//Stream Reliability Config
	ReliabilityConfig *Reliability
}

type Reliability struct {
	unordered        bool
	reliabilityType  byte
	reliabilityValue uint32
}

// Dial connects to the given network address and establishes a
// SCTP stream on top. The net.Conn in the config is ignored.
func (d *Dialer) Dial(network string, raddr *net.UDPAddr, streamIdentifier uint16) (*Stream, error) {
	if d.Config == nil {
		d.Config = &Config{
			LoggerFactory: logging.NewDefaultLoggerFactory(),
		}
	}
	a, err := DialAssociation(network, raddr, *d.Config)
	if err != nil {
		return nil, err
	}

	s, err := a.OpenStream(streamIdentifier, d.PayloadType)
	s.SetReliabilityParams(d.ReliabilityConfig.unordered, d.ReliabilityConfig.reliabilityType, d.ReliabilityConfig.reliabilityValue)
	return s, err
}
