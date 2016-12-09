package sockrus

import (
	"fmt"
	"net"
	"time"

	"github.com/Sirupsen/logrus"
	logrus_logstash "github.com/bshuster-repo/logrus-logstash-hook"
)

// Hook represents a connection to a socket
type Hook struct {
	formatter logrus_logstash.LogstashFormatter
	conn      net.Conn
	protocol  string
	address   string
	mute      bool
}

// NewHook establish a socket connection.
// Protocols allowed are: "udp", "tcp", "unix" (corresponds to SOCK_STREAM),
// "unixdomain" (corresponds to SOCK_DGRAM) or "unixpacket" (corresponds to SOCK_SEQPACKET).
//
// For TCP and UDP, address must have the form `host:port`.
//
// For Unix networks, the address must be a file system path.
func NewHook(protocol, address string) (*Hook, error) {
	logstashFormatter := logrus_logstash.LogstashFormatter{
		TimestampFormat: time.RFC3339Nano,
	}
	return &Hook{
		conn:      nil,
		protocol:  protocol,
		address:   address,
		formatter: logstashFormatter,
		mute:      false,
	}, nil
}

// Fire send log to the defined socket
func (h *Hook) Fire(entry *logrus.Entry) error {
	var err error
	if h.conn == nil {
		err = h.dialSock()
		if err != nil && h.mute == false {
			h.mute = true
			retErr := fmt.Errorf("Failed to dial. All further errors will be muted: %v", err)
			return retErr
		} else if err != nil && h.mute == true {
			return nil
		}
	}
	dataBytes, err := h.formatter.Format(entry)
	if err != nil {
		return err
	}
	if _, err = h.conn.Write(dataBytes); err != nil {
		_ = h.closeSock() // #nosec
		if h.mute == false {
			h.mute = true
			retErr := fmt.Errorf("Failed to write data. All further errors will be muted: %v", err)
			return retErr
		} else {
			return nil
		}
	}
	return nil
}

// Levels return an array of handled logging levels
func (h *Hook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// closeSock tries to close connection to Unix socket
func (h *Hook) closeSock() error {
	if h.conn == nil {
		return nil
	}
	err := h.conn.Close()
	h.conn = nil
	return err
}

// dialSock tries to connect to Unix socket
func (h *Hook) dialSock() error {
	conn, err := net.Dial(h.protocol, h.address)
	if err != nil {
		h.conn = nil
		return err
	}
	h.conn = conn
	h.mute = false
	return nil
}
