package cli

import (
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
)

type tailnetHost struct {
	bindHost    string
	displayHost string
}

type serveAddressOptions struct {
	addr              string
	port              int
	tailscale         bool
	addrExplicit      bool
	detectTailnetHost func() (tailnetHost, error)
	interfaceAddrs    func() ([]net.Addr, error)
}

type tailnetEnvironment struct {
	runCommand     func(string, ...string) ([]byte, error)
	interfaceAddrs func() ([]net.Addr, error)
}

func resolveServeAddress(opts serveAddressOptions) (string, string, error) {
	if opts.addr == "" {
		opts.addr = "127.0.0.1"
	}

	if opts.tailscale {
		if opts.addrExplicit {
			return "", "", fmt.Errorf("--tailscale cannot be used with --addr")
		}
		detect := opts.detectTailnetHost
		if detect == nil {
			detect = func() (tailnetHost, error) {
				return detectTailnetHost(defaultTailnetEnvironment())
			}
		}
		host, err := detect()
		if err != nil {
			return "", "", err
		}
		if host.displayHost == "" {
			host.displayHost = host.bindHost
		}
		return net.JoinHostPort(host.bindHost, strconv.Itoa(opts.port)), host.displayHost, nil
	}

	displayHost := opts.addr
	if isUnspecifiedHost(opts.addr) {
		displayHost = wildcardDisplayHost(opts.interfaceAddrs)
	}
	return net.JoinHostPort(opts.addr, strconv.Itoa(opts.port)), displayHost, nil
}

func defaultTailnetEnvironment() tailnetEnvironment {
	return tailnetEnvironment{
		runCommand: func(name string, args ...string) ([]byte, error) {
			return exec.Command(name, args...).Output()
		},
		interfaceAddrs: net.InterfaceAddrs,
	}
}

func detectTailnetHost(env tailnetEnvironment) (tailnetHost, error) {
	if env.runCommand == nil {
		env.runCommand = defaultTailnetEnvironment().runCommand
	}
	if env.interfaceAddrs == nil {
		env.interfaceAddrs = net.InterfaceAddrs
	}

	bindHost := ""
	if out, err := env.runCommand("tailscale", "ip", "-4"); err == nil {
		bindHost = firstTailnetIPv4(string(out))
	}
	if bindHost == "" {
		bindHost = firstTailnetInterfaceIPv4(env.interfaceAddrs)
	}
	if bindHost == "" {
		return tailnetHost{}, fmt.Errorf("detect Tailscale IPv4 address: no Tailscale IPv4 address found; run `tailscale up` or omit `--tailscale`")
	}

	displayHost := bindHost
	if out, err := env.runCommand("tailscale", "status", "--json"); err == nil {
		if dnsName := magicDNSName(out); dnsName != "" {
			displayHost = dnsName
		}
	}
	return tailnetHost{bindHost: bindHost, displayHost: displayHost}, nil
}

func firstTailnetIPv4(output string) string {
	for _, field := range strings.Fields(output) {
		ip := net.ParseIP(field)
		if isTailnetIPv4(ip) {
			return ip.String()
		}
	}
	return ""
}

func firstTailnetInterfaceIPv4(interfaceAddrs func() ([]net.Addr, error)) string {
	addrs, err := interfaceAddrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		ip := ipFromAddr(addr)
		if isTailnetIPv4(ip) {
			return ip.String()
		}
	}
	return ""
}

func wildcardDisplayHost(interfaceAddrs func() ([]net.Addr, error)) string {
	if interfaceAddrs == nil {
		interfaceAddrs = net.InterfaceAddrs
	}
	addrs, err := interfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, addr := range addrs {
		ip := ipFromAddr(addr)
		if isDisplayIPv4(ip) {
			return ip.String()
		}
	}
	return "127.0.0.1"
}

func isUnspecifiedHost(host string) bool {
	if host == "" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsUnspecified()
}

func isDisplayIPv4(ip net.IP) bool {
	ip4 := ip.To4()
	return ip4 != nil &&
		!ip4.IsLoopback() &&
		!ip4.IsUnspecified() &&
		!ip4.IsLinkLocalUnicast() &&
		(ip4.IsPrivate() || isTailnetIPv4(ip4))
}

func isTailnetIPv4(ip net.IP) bool {
	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}
	tailnet := net.IPNet{IP: net.IPv4(100, 64, 0, 0), Mask: net.CIDRMask(10, 32)}
	return tailnet.Contains(ip4)
}

func ipFromAddr(addr net.Addr) net.IP {
	switch typed := addr.(type) {
	case *net.IPNet:
		return typed.IP
	case *net.IPAddr:
		return typed.IP
	default:
		host, _, err := net.SplitHostPort(addr.String())
		if err == nil {
			return net.ParseIP(host)
		}
		return net.ParseIP(addr.String())
	}
}

func magicDNSName(data []byte) string {
	var status struct {
		Self struct {
			DNSName string `json:"DNSName"`
		} `json:"Self"`
	}
	if err := json.Unmarshal(data, &status); err != nil {
		return ""
	}
	name := strings.TrimSuffix(strings.TrimSpace(status.Self.DNSName), ".")
	if name == "" || strings.ContainsAny(name, " \t\r\n/") {
		return ""
	}
	return name
}
