// Package ooni queries the OONI aggregation API for real-world measurements
// of a domain from inside a country.
package ooni

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// DefaultBase is the public OONI API.
const DefaultBase = "https://api.ooni.io"

// Client talks to the OONI API.
type Client struct {
	Base string
	HTTP *http.Client
}

// Stats is the aggregated web_connectivity result for one domain.
type Stats struct {
	Domain       string `json:"domain"`
	Country      string `json:"country"`
	Since        string `json:"since"`
	Until        string `json:"until"`
	Measurements int    `json:"measurement_count"`
	Anomalies    int    `json:"anomaly_count"`
	Confirmed    int    `json:"confirmed_count"`
	Failures     int    `json:"failure_count"`
	OK           int    `json:"ok_count"`
	URL          string `json:"explorer_url"`
}

// Verdict summarises Stats in a few words.
func (s Stats) Verdict() string {
	usable := s.Measurements - s.Failures
	switch {
	case s.Measurements == 0:
		return "no data"
	case usable <= 0:
		return "inconclusive"
	}
	bad := float64(s.Anomalies+s.Confirmed) / float64(usable)
	switch {
	case s.Confirmed > 0 && bad >= 0.5:
		return "blocked"
	case bad >= 0.5:
		return "likely blocked"
	case bad >= 0.15:
		return "partially blocked"
	}
	return "accessible"
}

// Summary is a one-line human description.
func (s Stats) Summary() string {
	if s.Measurements == 0 {
		return fmt.Sprintf("no OONI measurements of %s from %s in %s..%s (it is not on the test lists)", s.Domain, s.Country, s.Since, s.Until)
	}
	usable := s.Measurements - s.Failures
	pct := 0
	if usable > 0 {
		pct = (s.Anomalies + s.Confirmed) * 100 / usable
	}
	return fmt.Sprintf("%s: %d measurements, %d%% anomalous, %d confirmed blocks (%s..%s)", s.Verdict(), s.Measurements, pct, s.Confirmed, s.Since, s.Until)
}

// Aggregate fetches stats for domain in country over the last days.
func (c *Client) Aggregate(ctx context.Context, country, domain string, days int, now time.Time) (Stats, error) {
	base := c.Base
	if base == "" {
		base = DefaultBase
	}
	hc := c.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 40 * time.Second}
	}
	if days <= 0 {
		days = 30
	}
	until := now.UTC().AddDate(0, 0, 1).Format("2006-01-02")
	since := now.UTC().AddDate(0, 0, -days).Format("2006-01-02")
	q := url.Values{"probe_cc": {country}, "domain": {domain}, "since": {since}, "until": {until}, "test_name": {"web_connectivity"}}
	req, _ := http.NewRequestWithContext(ctx, "GET", base+"/api/v1/aggregation?"+q.Encode(), nil)
	req.Header.Set("User-Agent", "marz (+https://github.com/mrzroot/marz)")
	resp, err := hc.Do(req)
	if err != nil {
		return Stats{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return Stats{}, fmt.Errorf("OONI API: HTTP %d", resp.StatusCode)
	}
	var body struct {
		Result Stats `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return Stats{}, fmt.Errorf("OONI API: %w", err)
	}
	s := body.Result
	s.Domain, s.Country, s.Since, s.Until = domain, country, since, until
	ex := url.Values{"probe_cc": {country}, "domain": {domain}, "since": {since}, "until": {until}, "test_name": {"web_connectivity"}, "axis_x": {"measurement_start_day"}}
	s.URL = "https://explorer.ooni.org/chart/mat?" + ex.Encode()
	return s, nil
}
