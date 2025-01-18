package congestion

import (
	"github.com/v2fly/hysteria/core/v2/international/congestion/bbr"
	"github.com/v2fly/hysteria/core/v2/international/congestion/brutal"
	"github.com/apernet/quic-go"
)

func UseBBR(conn quic.Connection) {
	conn.SetCongestionControl(bbr.NewBbrSender(
		bbr.DefaultClock{},
		bbr.GetInitialPacketSize(conn.RemoteAddr()),
	))
}

func UseBrutal(conn quic.Connection, tx uint64) {
	conn.SetCongestionControl(brutal.NewBrutalSender(tx))
}
