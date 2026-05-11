package cli

import (
	"errors"
	"net"
	"strings"
	"testing"
)

func TestResolveServeAddressTailscaleUsesDetectedHost(t *testing.T) {
	listenAddr, displayHost, err := resolveServeAddress(serveAddressOptions{
		addr:      "127.0.0.1",
		port:      8080,
		tailscale: true,
		detectTailnetHost: func() (tailnetHost, error) {
			return tailnetHost{bindHost: "100.89.157.2", displayHost: "ubuntu.tailnet.ts.net"}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if listenAddr != "100.89.157.2:8080" {
		t.Fatalf("listen address = %q", listenAddr)
	}
	if displayHost != "ubuntu.tailnet.ts.net" {
		t.Fatalf("display host = %q", displayHost)
	}
}

func TestResolveServeAddressRejectsTailscaleWithAddr(t *testing.T) {
	_, _, err := resolveServeAddress(serveAddressOptions{
		addr:         "0.0.0.0",
		addrExplicit: true,
		tailscale:    true,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--tailscale cannot be used with --addr") {
		t.Fatalf("error = %q", err)
	}
}

func TestResolveServeAddressWildcardUsesPrivateDisplayHost(t *testing.T) {
	listenAddr, displayHost, err := resolveServeAddress(serveAddressOptions{
		addr: "0.0.0.0",
		port: 18084,
		interfaceAddrs: func() ([]net.Addr, error) {
			return []net.Addr{
				ipNet("127.0.0.1", 32),
				ipNet("192.168.12.34", 24),
			}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if listenAddr != "0.0.0.0:18084" {
		t.Fatalf("listen address = %q", listenAddr)
	}
	if displayHost != "192.168.12.34" {
		t.Fatalf("display host = %q", displayHost)
	}
}

func TestResolveServeAddressWildcardFallsBackToLoopbackDisplayHost(t *testing.T) {
	_, displayHost, err := resolveServeAddress(serveAddressOptions{
		addr: "0.0.0.0",
		port: 18084,
		interfaceAddrs: func() ([]net.Addr, error) {
			return []net.Addr{ipNet("127.0.0.1", 32)}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if displayHost != "127.0.0.1" {
		t.Fatalf("display host = %q", displayHost)
	}
}

func TestResolveServeAddressWildcardCanUseTailnetDisplayHost(t *testing.T) {
	_, displayHost, err := resolveServeAddress(serveAddressOptions{
		addr: "0.0.0.0",
		port: 18084,
		interfaceAddrs: func() ([]net.Addr, error) {
			return []net.Addr{
				ipNet("127.0.0.1", 32),
				ipNet("100.89.157.2", 32),
			}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if displayHost != "100.89.157.2" {
		t.Fatalf("display host = %q", displayHost)
	}
}

func TestDetectTailnetHostUsesTailscaleCommandAndMagicDNS(t *testing.T) {
	host, err := detectTailnetHost(tailnetEnvironment{
		runCommand: func(name string, args ...string) ([]byte, error) {
			command := name + " " + strings.Join(args, " ")
			switch command {
			case "tailscale ip -4":
				return []byte("100.89.157.2\n"), nil
			case "tailscale status --json":
				return []byte(`{"Self":{"DNSName":"ubuntu.tailnet.ts.net."}}`), nil
			default:
				return nil, errors.New("unexpected command")
			}
		},
		interfaceAddrs: func() ([]net.Addr, error) {
			return nil, errors.New("interfaces should not be used")
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if host.bindHost != "100.89.157.2" {
		t.Fatalf("bind host = %q", host.bindHost)
	}
	if host.displayHost != "ubuntu.tailnet.ts.net" {
		t.Fatalf("display host = %q", host.displayHost)
	}
}

func TestDetectTailnetHostFallsBackToInterfaceAddress(t *testing.T) {
	host, err := detectTailnetHost(tailnetEnvironment{
		runCommand: func(string, ...string) ([]byte, error) {
			return nil, errors.New("tailscale unavailable")
		},
		interfaceAddrs: func() ([]net.Addr, error) {
			return []net.Addr{
				ipNet("10.0.0.8", 24),
				ipNet("100.89.157.2", 32),
			}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if host.bindHost != "100.89.157.2" {
		t.Fatalf("bind host = %q", host.bindHost)
	}
	if host.displayHost != "100.89.157.2" {
		t.Fatalf("display host = %q", host.displayHost)
	}
}

func TestDetectTailnetHostFailsClearly(t *testing.T) {
	_, err := detectTailnetHost(tailnetEnvironment{
		runCommand: func(string, ...string) ([]byte, error) {
			return nil, errors.New("tailscale unavailable")
		},
		interfaceAddrs: func() ([]net.Addr, error) {
			return []net.Addr{ipNet("127.0.0.1", 32)}, nil
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "Tailscale IPv4 address") {
		t.Fatalf("error = %q", err)
	}
}

func ipNet(ip string, bits int) *net.IPNet {
	parsed := net.ParseIP(ip)
	if parsed.To4() != nil {
		return &net.IPNet{IP: parsed, Mask: net.CIDRMask(bits, 32)}
	}
	return &net.IPNet{IP: parsed, Mask: net.CIDRMask(bits, 128)}
}
