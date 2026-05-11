package cli

import (
	"errors"
	"net"
	"strings"
	"testing"
)

func TestResolveServeAddressTailscaleUsesDetectedHost(t *testing.T) {
	serveAddr, err := resolveServeAddress(serveAddressOptions{
		addr:      "127.0.0.1",
		port:      8080,
		tailscale: true,
		detectTailnetHost: func() (tailnetHost, error) {
			return tailnetHost{bindHost: "100.89.157.2"}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if serveAddr.listenAddr != "100.89.157.2:8080" {
		t.Fatalf("listen address = %q", serveAddr.listenAddr)
	}
	if got, want := serveAddr.displayHosts, []string{"100.89.157.2"}; !stringSlicesEqual(got, want) {
		t.Fatalf("display hosts = %q, want %q", got, want)
	}
}

func TestResolveServeAddressRejectsTailscaleWithAddr(t *testing.T) {
	_, err := resolveServeAddress(serveAddressOptions{
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
	serveAddr, err := resolveServeAddress(serveAddressOptions{
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
	if serveAddr.listenAddr != "0.0.0.0:18084" {
		t.Fatalf("listen address = %q", serveAddr.listenAddr)
	}
	if got, want := serveAddr.displayHosts, []string{"192.168.12.34"}; !stringSlicesEqual(got, want) {
		t.Fatalf("display hosts = %q, want %q", got, want)
	}
}

func TestResolveServeAddressWildcardFallsBackToLoopbackDisplayHost(t *testing.T) {
	serveAddr, err := resolveServeAddress(serveAddressOptions{
		addr: "0.0.0.0",
		port: 18084,
		interfaceAddrs: func() ([]net.Addr, error) {
			return []net.Addr{ipNet("127.0.0.1", 32)}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := serveAddr.displayHosts, []string{"127.0.0.1"}; !stringSlicesEqual(got, want) {
		t.Fatalf("display hosts = %q, want %q", got, want)
	}
}

func TestResolveServeAddressWildcardCanUseTailnetDisplayHost(t *testing.T) {
	serveAddr, err := resolveServeAddress(serveAddressOptions{
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
	if got, want := serveAddr.displayHosts, []string{"100.89.157.2"}; !stringSlicesEqual(got, want) {
		t.Fatalf("display hosts = %q, want %q", got, want)
	}
}

func TestDetectTailnetHostUsesTailscaleCommand(t *testing.T) {
	host, err := detectTailnetHost(tailnetEnvironment{
		runCommand: func(name string, args ...string) ([]byte, error) {
			command := name + " " + strings.Join(args, " ")
			switch command {
			case "tailscale ip -4":
				return []byte("100.89.157.2\n"), nil
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

func stringSlicesEqual(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
