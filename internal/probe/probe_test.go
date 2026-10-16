package probe

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func tlsServer(t *testing.T, h http.HandlerFunc) (*Prober, func()) {
	srv := httptest.NewTLSServer(h)
	pool := x509.NewCertPool()
	pool.AddCert(srv.Certificate())
	_, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
	p := &Prober{
		Timeout:      3 * time.Second,
		Port:         port,
		AllowPrivate: true,
		TLS:          &tls.Config{RootCAs: pool, ServerName: "example.com"},
		Resolve:      func(context.Context, string) ([]string, error) { return []string{"127.0.0.1"}, nil },
	}
	return p, srv.Close
}

func TestOK(t *testing.T) {
	p, done := tlsServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Errorf("path %q", r.URL.Path)
		}
		w.WriteHeader(401)
	})
	defer done()
	r := p.Probe(context.Background(), "api.openai.com")
	if r.Verdict != OK || r.Status != 401 {
		t.Fatalf("%+v", r)
	}
	if len(r.Steps) != 4 {
		t.Fatalf("steps %+v", r.Steps)
	}
}

func TestFingerprints(t *testing.T) {
	cases := []struct {
		status int
		body   string
		want   string
		detail string
	}{
		{403, `<title>Access denied</title> error code: 1009`, GeoBlocked, "Cloudflare 1009"},
		{403, `The Amazon CloudFront distribution is configured to block access from your country.`, GeoBlocked, "CloudFront"},
		{403, `{"error":{"code":"unsupported_country_region_territory"}}`, GeoBlocked, "OpenAI"},
		{400, `{"error":{"message":"User location is not supported for the API use."}}`, GeoBlocked, "Google"},
		{403, `Since Docker is a US company, we must comply with US export control regulations.`, GeoBlocked, "export-control"},
		{200, `<iframe src="http://10.10.34.34?type=Invalid Site"></iframe>`, HTTPBlocked, "Iranian filter"},
		{403, `Forbidden`, GeoBlocked, "HTTP 403"},
		{200, `hello`, OK, "reachable"},
	}
	for _, c := range cases {
		p, done := tlsServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(c.status)
			w.Write([]byte(c.body))
		})
		r := p.Probe(context.Background(), "https://example.com/")
		done()
		if r.Verdict != c.want || !strings.Contains(r.Detail, c.detail) {
			t.Errorf("%d %q: got %s / %q", c.status, c.body, r.Verdict, r.Detail)
		}
	}
}

func TestIranDNS(t *testing.T) {
	p := &Prober{Resolve: func(context.Context, string) ([]string, error) { return []string{"10.10.34.35"}, nil }}
	r := p.Probe(context.Background(), "www.youtube.com")
	if r.Verdict != DNSBlocked || !strings.Contains(r.Detail, "Iranian filter") {
		t.Fatalf("%+v", r)
	}
}

func TestPrivateDNS(t *testing.T) {
	p := &Prober{Resolve: func(context.Context, string) ([]string, error) { return []string{"192.168.1.9"}, nil }}
	if r := p.Probe(context.Background(), "x.com"); r.Verdict != DNSBlocked {
		t.Fatalf("%+v", r)
	}
}

func TestTLSReset(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			buf := make([]byte, 512)
			c.Read(buf) // read ClientHello, then reset like a DPI box
			c.(*net.TCPConn).SetLinger(0)
			c.Close()
		}
	}()
	_, port, _ := net.SplitHostPort(ln.Addr().String())
	p := &Prober{Port: port, AllowPrivate: true, Timeout: 2 * time.Second,
		Resolve: func(context.Context, string) ([]string, error) { return []string{"127.0.0.1"}, nil }}
	r := p.Probe(context.Background(), "blocked.example")
	if r.Verdict != TLSBlocked {
		t.Fatalf("%+v", r)
	}
}

func TestTCPRefused(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	_, port, _ := net.SplitHostPort(ln.Addr().String())
	ln.Close()
	p := &Prober{Port: port, AllowPrivate: true, Timeout: time.Second,
		Resolve: func(context.Context, string) ([]string, error) { return []string{"127.0.0.1"}, nil }}
	if r := p.Probe(context.Background(), "down.example"); r.Verdict != TCPBlocked {
		t.Fatalf("%+v", r)
	}
}

func TestBadCert(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	_, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
	p := &Prober{Port: port, AllowPrivate: true, Timeout: 2 * time.Second,
		Resolve: func(context.Context, string) ([]string, error) { return []string{"127.0.0.1"}, nil }}
	if r := p.Probe(context.Background(), "fonts.googleapis.com"); r.Verdict != TLSInvalid {
		t.Fatalf("%+v", r)
	}
}

func TestSplitTarget(t *testing.T) {
	cases := map[string][2]string{
		"https://fonts.googleapis.com/css2?family=Inter": {"fonts.googleapis.com", "/css2?family=Inter"},
		"API.OPENAI.COM":     {"api.openai.com", "/v1/models"},
		"example.com:8443/x": {"example.com", "/x"},
		"example.com?q=1":    {"example.com", "/?q=1"},
		"//cdn.jsdelivr.net": {"cdn.jsdelivr.net", "/"},
	}
	for in, want := range cases {
		h, p := splitTarget(in)
		if h != want[0] || p != want[1] {
			t.Errorf("%s: %s %s", in, h, p)
		}
	}
}

func TestNeedsKeyIsOK(t *testing.T) {
	p, done := tlsServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		w.Write([]byte(`{"error":{"code":403,"message":"Method doesn't allow unregistered callers (callers without established identity). Please use API Key"}}`))
	})
	defer done()
	if r := p.Probe(context.Background(), "generativelanguage.googleapis.com"); r.Verdict != OK {
		t.Fatalf("%+v", r)
	}
}
