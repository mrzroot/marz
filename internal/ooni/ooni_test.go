package ooni

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAggregate(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Write([]byte(`{"v":0,"result":{"anomaly_count":884,"confirmed_count":3173,"failure_count":341,"ok_count":268,"measurement_count":4666}}`))
	}))
	defer srv.Close()
	c := &Client{Base: srv.URL}
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	s, err := c.Aggregate(context.Background(), "IR", "www.youtube.com", 30, now)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"probe_cc=IR", "domain=www.youtube.com", "since=2026-09-16", "until=2026-10-17", "test_name=web_connectivity"} {
		if !strings.Contains(gotQuery, want) {
			t.Errorf("query %q missing %q", gotQuery, want)
		}
	}
	if s.Verdict() != "blocked" {
		t.Errorf("verdict %q", s.Verdict())
	}
	if !strings.Contains(s.Summary(), "93% anomalous") {
		t.Errorf("summary %q", s.Summary())
	}
	if !strings.HasPrefix(s.URL, "https://explorer.ooni.org/chart/mat?") {
		t.Errorf("url %q", s.URL)
	}
}

func TestVerdicts(t *testing.T) {
	cases := []struct {
		s    Stats
		want string
	}{
		{Stats{}, "no data"},
		{Stats{Measurements: 10, Failures: 10}, "inconclusive"},
		{Stats{Measurements: 100, OK: 95, Anomalies: 5}, "accessible"},
		{Stats{Measurements: 100, OK: 70, Anomalies: 30}, "partially blocked"},
		{Stats{Measurements: 100, OK: 20, Anomalies: 80}, "likely blocked"},
	}
	for _, c := range cases {
		if got := c.s.Verdict(); got != c.want {
			t.Errorf("%+v: %q want %q", c.s, got, c.want)
		}
	}
}

func TestHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(503) }))
	defer srv.Close()
	_, err := (&Client{Base: srv.URL}).Aggregate(context.Background(), "IR", "x.com", 7, time.Now())
	if err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("err = %v", err)
	}
}
