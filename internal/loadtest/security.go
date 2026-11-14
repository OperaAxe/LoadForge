package loadtest

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var blockedCIDRs = mustParseCIDRs([]string{
	"0.0.0.0/8", "10.0.0.0/8", "100.64.0.0/10", "127.0.0.0/8",
	"169.254.0.0/16", "172.16.0.0/12", "192.0.0.0/24", "192.0.2.0/24",
	"192.168.0.0/16", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24",
	"224.0.0.0/4", "240.0.0.0/4",
	"::/128", "::1/128", "fc00::/7", "fe80::/10", "ff00::/8", "2001:db8::/32",
})

func mustParseCIDRs(values []string) []*net.IPNet {
	out := make([]*net.IPNet, 0, len(values))
	for _, value := range values {
		_, block, err := net.ParseCIDR(value)
		if err != nil { panic(err) }
		out = append(out, block)
	}
	return out
}

func isBlockedIP(ip net.IP) bool {
	if ip == nil || !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	if v4 := ip.To4(); v4 != nil { ip = v4 }
	for _, block := range blockedCIDRs {
		if block.Contains(ip) { return true }
	}
	return false
}

func parseAndValidateURL(raw string, allowPrivate bool) (*url.URL, error) {
	u, err := url.ParseRequestURI(strings.TrimSpace(raw))
	if err != nil { return nil, fmt.Errorf("invalid target URL: %w", err) }
	if u.Scheme != "http" && u.Scheme != "https" { return nil, fmt.Errorf("only http and https URLs are supported") }
	if u.User != nil { return nil, fmt.Errorf("URL credentials are not allowed") }
	if u.Hostname() == "" { return nil, fmt.Errorf("target URL must include a hostname") }
	if strings.Contains(u.Hostname(), "%") { return nil, fmt.Errorf("IPv6 zone identifiers are not allowed") }
	if port := u.Port(); port != "" {
		p, err := strconv.Atoi(port)
		if err != nil || p < 1 || p > 65535 { return nil, fmt.Errorf("target URL contains an invalid port") }
	}
	host := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	if !allowPrivate && (host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") || host == "metadata.google.internal") {
		return nil, fmt.Errorf("local and metadata hostnames are blocked by default")
	}
	return u, nil
}

func resolveAndCheck(ctx context.Context, host string, allowPrivate bool) ([]net.IPAddr, error) {
	if ip := net.ParseIP(host); ip != nil {
		if !allowPrivate && isBlockedIP(ip) { return nil, fmt.Errorf("target resolves to a private, reserved, or non-routable IP address") }
		return []net.IPAddr{{IP: ip}}, nil
	}
	lookupCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	ips, err := net.DefaultResolver.LookupIPAddr(lookupCtx, host)
	if err != nil { return nil, fmt.Errorf("could not resolve target hostname: %w", err) }
	if len(ips) == 0 { return nil, fmt.Errorf("target hostname returned no IP addresses") }
	if !allowPrivate {
		for _, item := range ips {
			if isBlockedIP(item.IP) { return nil, fmt.Errorf("target hostname resolves to a private, reserved, or non-routable IP address") }
		}
	}
	return ips, nil
}

func validateTarget(ctx context.Context, raw string, allowPrivate bool) (*url.URL, error) {
	u, err := parseAndValidateURL(raw, allowPrivate)
	if err != nil { return nil, err }
	if _, err := resolveAndCheck(ctx, u.Hostname(), allowPrivate); err != nil { return nil, err }
	return u, nil
}

func safeDialContext(allowPrivate bool) func(context.Context, string, string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil { return nil, err }
		ips, err := resolveAndCheck(ctx, host, allowPrivate)
		if err != nil { return nil, err }
		var lastErr error
		for _, item := range ips {
			conn, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(item.IP.String(), port))
			if dialErr == nil { return conn, nil }
			lastErr = dialErr
		}
		if lastErr == nil { lastErr = fmt.Errorf("no usable target IP address") }
		return nil, lastErr
	}
}
