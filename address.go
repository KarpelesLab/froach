package froach

import (
	"net"

	"github.com/KarpelesLab/fleet"
)

func getAddrs() []string {
	peers := fleet.Self().GetPeers()
	var final []string

	for _, v := range peers {
		addr := addrIP(v.Addr())
		if addr == nil {
			continue
		}
		ipport := &net.TCPAddr{IP: addr, Port: 36257}
		final = append(final, ipport.String())
	}

	return final
}

func addrIP(a net.Addr) net.IP {
	switch b := a.(type) {
	case *net.TCPAddr:
		return b.IP
	case *net.UDPAddr:
		return b.IP
	case *net.IPAddr:
		return b.IP
	default:
		return nil
	}
	return nil
}
