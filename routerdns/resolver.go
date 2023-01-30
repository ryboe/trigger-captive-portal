package routerdns

import (
	"context"
	"net"
)

// Resolver returns a net.Resolver that satisfies the go-socks5 Resolver
// interface.
type Resolver struct {
	*net.Resolver
}

// NewResolver returns a new Resolver that uses the given IP and Dialer for DNS
// lookups.
func NewResolver(dnsIP string, dialer *net.Dialer) *Resolver {
	const dnsPort = "53"
	dnsAddr := net.JoinHostPort(dnsIP, dnsPort)

	return &Resolver{
		Resolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				// Redirect all Resolver dials to the router DNS server.
				return dialer.DialContext(ctx, network, dnsAddr)
			},
		},
	}
}

// Resolve returns the first IPv4 address for the given domain name using the
// upstream router's DNS resolver.
func (r *Resolver) Resolve(ctx context.Context, name string) (context.Context, net.IP, error) {
	addrs, err := r.LookupIPAddr(ctx, name)
	if err != nil {
		return ctx, nil, err
	}
	if len(addrs) == 0 {
		return ctx, nil, nil
	}

	// go-socks5 requires IPv4 addresses :(
	for _, addr := range addrs {
		if addr.IP.To4() != nil {
			return ctx, addr.IP, nil
		}
	}

	return ctx, addrs[0].IP, nil
}
