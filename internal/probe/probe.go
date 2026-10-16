// Package probe tests reachability of a host from the current network, step
// by step (DNS, TCP, TLS, HTTP), and recognises common block and geo-block
// fingerprints.
package probe

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"syscall"
	"time"
)

// Verdicts.
const (
	OK          = "ok"
	DNSBlocked  = "dns-blocked"
	DNSFail     = "dns-failed"
	TCPBlocked  = "tcp-blocked"
	TLSBlocked  = "tls-blocked"
	TLSInvalid  = "tls-invalid"
	HTTPBlocked = "http-blocked"
	GeoBlocked  = "geo-blocked"
	HTTPError   = "http-error"
)

// Result of probing one host.
type Result struct {
	Host    string        `json:"host"`
	URL     string        `json:"url"`
	IPs     []string      `json:"ips,omitempty"`
	Verdict string        `json:"verdict"`
	Detail  string        `json:"detail"`
	Status  int           `json:"http_status,omitempty"`
	Server  string        `json:"server,omitempty"`
	Steps   []Step        `json:"steps"`
	Elapsed time.Duration `json:"elapsed_ns"`
}

// Step is one stage of the probe.
type Step struct {
	Name string        `json:"name"`
	OK   bool          `json:"ok"`
	Info string        `json:"info"`
	Took time.Duration `json:"took_ns"`
}

// Prober runs probes. Zero value is usable.
type Prober struct {
	Timeout time.Duration
	// Resolve overrides DNS (tests).
	Resolve func(ctx context.Context, host string) ([]string, error)
	// Port overrides 443 (tests).
	Port string
	// TLS is cloned for each handshake; nil means system roots.
	TLS *tls.Config
	// AllowPrivate disables the private-address DNS check (tests).
	AllowPrivate bool
}

// Paths that give a meaningful answer for API hosts.
var probePath = map[string]string{
	"api.openai.com":                    "/v1/models",
	"api.anthropic.com":                 "/v1/models",
	"generativelanguage.googleapis.com": "/v1beta/models",
	"fonts.googleapis.com":              "/css2?family=Roboto",
	"maps.googleapis.com":               "/maps/api/js",
	"firestore.googleapis.com":          "/",
	"registry-1.docker.io":              "/v2/",
	"pub.dev":                           "/api/packages/http",
}

// Normal "you need credentials" answers: the service is serving this network.
var authRe = regexp.MustCompile(`(?i)unregistered callers|api key|api-key header|authentication_error|invalid_api_key|missing bearer|unauthorized`)

// Proxies and VPN apps in fake-IP mode answer DNS from 198.18.0.0/15.
var fakeIPNet = mustCIDR("198.18.0.0/15")

// Iran's filter answers DNS with addresses in 10.10.34.0/24.
var iranFilterNet = mustCIDR("10.10.34.0/24")

var privateNets = []*net.IPNet{mustCIDR("10.0.0.0/8"), mustCIDR("172.16.0.0/12"), mustCIDR("192.168.0.0/16"), mustCIDR("127.0.0.0/8"), mustCIDR("0.0.0.0/8"), mustCIDR("::1/128"), mustCIDR("fc00::/7")}

func mustCIDR(s string) *net.IPNet {
	_, n, err := net.ParseCIDR(s)
	if err != nil {
		panic(err)
	}
	return n
}

type fingerprint struct {
	re      *regexp.Regexp
	verdict string
	detail  string
}

var fingerprints = []fingerprint{
	{regexp.MustCompile(`(?i)peyvandha\.ir|10\.10\.34\.3[456]`), HTTPBlocked, "Iranian filter page (peyvandha)"},
	{regexp.MustCompile(`(?i)warning\.rt\.ru|blocklist\.rkn\.gov\.ru|eais\.rkn\.gov\.ru`), HTTPBlocked, "Russian RKN block page"},
	{regexp.MustCompile(`(?i)error code:? ?1009|1009</span>|has banned the country or region`), GeoBlocked, "Cloudflare 1009: the site owner blocks your country"},
	{regexp.MustCompile(`(?i)configured to block access from your country`), GeoBlocked, "CloudFront geo-restriction"},
	{regexp.MustCompile(`(?i)unsupported_country_region_territory|country, region, or territory not supported`), GeoBlocked, "OpenAI: unsupported country"},
	{regexp.MustCompile(`(?i)user location is not supported`), GeoBlocked, "Google: user location is not supported"},
	{regexp.MustCompile(`(?i)export control|embargoed|sanction`), GeoBlocked, "refused for export-control / sanctions reasons"},
	{regexp.MustCompile(`(?i)not (?:available|supported) in your (?:country|region)|unavailable in your (?:country|region)|from your (?:country|region|location)`), GeoBlocked, "service says it is not available in your region"},
}

// Probe tests one host (or URL).
func (p *Prober) Probe(ctx context.Context, target string) Result {
	start := time.Now()
	timeout := p.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	host, path := splitTarget(target)
	port := p.Port
	if port == "" {
		port = "443"
	}
	res := Result{Host: host, URL: "https://" + host + path}
	defer func() { res.Elapsed = time.Since(start) }()
	step := func(name string, ok bool, t0 time.Time, f string, a ...any) {
		res.Steps = append(res.Steps, Step{Name: name, OK: ok, Info: fmt.Sprintf(f, a...), Took: time.Since(t0)})
	}

	// DNS
	t0 := time.Now()
	dctx, cancel := context.WithTimeout(ctx, timeout)
	ips, err := p.resolve(dctx, host)
	cancel()
	if err != nil {
		step("dns", false, t0, "%v", err)
		res.Verdict, res.Detail = DNSFail, "DNS lookup failed: "+short(err)
		return res
	}
	res.IPs = ips
	for _, s := range ips {
		ip := net.ParseIP(s)
		if ip == nil {
			continue
		}
		if iranFilterNet.Contains(ip) {
			step("dns", false, t0, "%s", strings.Join(ips, ", "))
			res.Verdict, res.Detail = DNSBlocked, "DNS answered "+s+" (Iranian filter address)"
			return res
		}
		if !p.AllowPrivate {
			for _, n := range privateNets {
				if n.Contains(ip) {
					step("dns", false, t0, "%s", strings.Join(ips, ", "))
					res.Verdict, res.Detail = DNSBlocked, "DNS answered a private address "+s+" (poisoned or sinkholed)"
					return res
				}
			}
		}
	}
	info := strings.Join(ips, ", ")
	if ip := net.ParseIP(ips[0]); ip != nil && fakeIPNet.Contains(ip) {
		info += " (fake-IP: a local proxy/VPN answers DNS, so results describe that proxy)"
	}
	step("dns", true, t0, "%s", info)

	// TCP
	t0 = time.Now()
	var conn net.Conn
	var ip string
	var lastErr error
	for _, cand := range ips {
		d := net.Dialer{Timeout: timeout}
		conn, lastErr = d.DialContext(ctx, "tcp", net.JoinHostPort(cand, port))
		if lastErr == nil {
			ip = cand
			break
		}
	}
	if conn == nil {
		step("tcp", false, t0, "%v", lastErr)
		res.Verdict, res.Detail = TCPBlocked, "TCP connect to port "+port+" failed: "+short(lastErr)
		return res
	}
	step("tcp", true, t0, "%s:%s", ip, port)

	// TLS
	t0 = time.Now()
	cfg := p.tlsConfig(host)
	tc := tls.Client(conn, cfg)
	tc.SetDeadline(time.Now().Add(timeout))
	err = tc.HandshakeContext(ctx)
	tc.Close()
	if err != nil {
		step("tls", false, t0, "%v", err)
		var cv *tls.CertificateVerificationError
		switch {
		case errors.As(err, &cv) || strings.Contains(err.Error(), "certificate"):
			res.Verdict, res.Detail = TLSInvalid, "certificate is not valid for "+host+" (interception?): "+short(err)
		case isReset(err):
			res.Verdict, res.Detail = TLSBlocked, "connection reset during the TLS handshake (SNI filtering)"
		case isTimeout(err):
			res.Verdict, res.Detail = TLSBlocked, "TLS handshake timed out (SNI filtering or throttling)"
		default:
			res.Verdict, res.Detail = TLSBlocked, "TLS handshake failed: "+short(err)
		}
		return res
	}
	step("tls", true, t0, "%s", tlsVersion(tc.ConnectionState().Version))

	// HTTP
	t0 = time.Now()
	tr := &http.Transport{
		Proxy:           nil,
		TLSClientConfig: p.tlsConfig(host),
		DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			d := net.Dialer{Timeout: timeout}
			return d.DialContext(ctx, network, net.JoinHostPort(ip, port))
		},
		ForceAttemptHTTP2:     true,
		ResponseHeaderTimeout: timeout,
	}
	defer tr.CloseIdleConnections()
	hc := &http.Client{Transport: tr, Timeout: 2 * timeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	req, _ := http.NewRequestWithContext(ctx, "GET", res.URL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0 Safari/537.36 marz-probe")
	resp, err := hc.Do(req)
	if err != nil {
		step("http", false, t0, "%v", err)
		if isReset(err) || isTimeout(err) {
			res.Verdict, res.Detail = HTTPBlocked, "HTTP request was cut: "+short(err)
		} else {
			res.Verdict, res.Detail = HTTPError, short(err)
		}
		return res
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256<<10))
	resp.Body.Close()
	res.Status, res.Server = resp.StatusCode, resp.Header.Get("Server")
	loc := resp.Header.Get("Location")
	step("http", resp.StatusCode < 400 || resp.StatusCode == 401, t0, "HTTP %d %s", resp.StatusCode, res.Server)
	text := string(body) + "\n" + loc
	for _, f := range fingerprints {
		if f.re.MatchString(text) && (resp.StatusCode >= 300 || f.verdict == HTTPBlocked) {
			res.Verdict, res.Detail = f.verdict, f.detail+fmt.Sprintf(" (HTTP %d)", resp.StatusCode)
			return res
		}
	}
	if authRe.MatchString(text) {
		res.Verdict, res.Detail = OK, fmt.Sprintf("reachable (HTTP %d: the API wants a key, which means it accepted this network)", resp.StatusCode)
		return res
	}
	switch {
	case resp.StatusCode == 403 || resp.StatusCode == 451:
		res.Verdict, res.Detail = GeoBlocked, fmt.Sprintf("HTTP %d from %s; often a country block. Compare with a VPN.", resp.StatusCode, serverName(res.Server))
	default:
		res.Verdict = OK
		res.Detail = fmt.Sprintf("reachable (HTTP %d)", resp.StatusCode)
		if resp.StatusCode == 401 || resp.StatusCode == 404 || resp.StatusCode == 405 {
			res.Detail += "; the server answered normally"
		}
	}
	return res
}

func serverName(s string) string {
	if s == "" {
		return "the server"
	}
	return s
}

func (p *Prober) resolve(ctx context.Context, host string) ([]string, error) {
	if p.Resolve != nil {
		return p.Resolve(ctx, host)
	}
	if ip := net.ParseIP(host); ip != nil {
		return []string{host}, nil
	}
	return net.DefaultResolver.LookupHost(ctx, host)
}

func (p *Prober) tlsConfig(host string) *tls.Config {
	var c *tls.Config
	if p.TLS != nil {
		c = p.TLS.Clone()
	} else {
		c = &tls.Config{}
	}
	if c.ServerName == "" {
		c.ServerName = host
	}
	return c
}

func splitTarget(t string) (host, path string) {
	t = strings.TrimSpace(t)
	t = strings.TrimPrefix(strings.TrimPrefix(t, "https://"), "http://")
	t = strings.TrimPrefix(t, "//")
	if i := strings.IndexAny(t, "/?"); i >= 0 {
		host, path = t[:i], t[i:]
		if path[0] == '?' {
			path = "/" + path
		}
	} else {
		host = t
	}
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	if path == "" {
		path = probePath[host]
		if path == "" {
			path = "/"
		}
	}
	return
}

func isReset(err error) bool {
	return errors.Is(err, syscall.ECONNRESET) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) ||
		strings.Contains(err.Error(), "connection reset") || strings.Contains(err.Error(), "forcibly closed")
}

func isTimeout(err error) bool {
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout() || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, os.ErrDeadlineExceeded)
}

func short(err error) string {
	s := err.Error()
	if len(s) > 160 {
		s = s[:160] + "…"
	}
	return s
}

func tlsVersion(v uint16) string {
	switch v {
	case tls.VersionTLS13:
		return "TLS 1.3"
	case tls.VersionTLS12:
		return "TLS 1.2"
	}
	return fmt.Sprintf("TLS 0x%x", v)
}

// Where asks Cloudflare which country the current network exits from.
func Where(ctx context.Context) (country, ip string, err error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://www.cloudflare.com/cdn-cgi/trace", nil)
	resp, err := (&http.Client{Transport: &http.Transport{Proxy: nil}}).Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	sc := bufio.NewScanner(io.LimitReader(resp.Body, 8<<10))
	for sc.Scan() {
		k, v, _ := strings.Cut(sc.Text(), "=")
		switch k {
		case "loc":
			country = v
		case "ip":
			ip = v
		}
	}
	return country, ip, nil
}
