package freedns

import (
	"github.com/miekg/dns"
	"testing"
)

func TestAppendDefaultPort(t *testing.T) {
	cases := []struct {
		i string
		o string
	}{
		{"127.0.0.1", "127.0.0.1:53"},
		{"114.114.114.114:5353", "114.114.114.114:5353"},
		{"::1", "::1"},
	}
	for _, c := range cases {
		if appendDefaultPort(c.i) != c.o {
			t.Errorf("Expected: %s", c.o)
		}
	}
}

func TestSmokingNewRunAndShutdown(t *testing.T) {
	// new the server
	s, err := NewServer(Config{
		FastDNS:  "114.114.114.114",
		CleanDNS: "8.8.8.8",
		Listen:   "127.0.0.1:52345",
		CacheCap: 1024 * 5,
	})
	if err != nil {
		t.Error(err)
	}

	// run the server
	shut := make(chan bool, 2)
	go func() {
		err := s.Run()
		if err != nil {
			t.Error(err)
		}
		_ = <-shut
		s.Shutdown()
	}()

	tests := []struct {
		domain           string
		qtype            uint16
		net              string
		expectedUpstream string
	}{
		{"ustc.edu.cn.", dns.TypeMX, "udp", "8.8.8.8:53"},
		{"ustc.edu.cn.", dns.TypeA, "udp", "114.114.114.114:53"},
		{"ustc.edu.cn.", dns.TypeMX, "udp", "114.114.114.114:53"},
		{"google.com.", dns.TypeA, "udp", "8.8.8.8:53"},
		{"mi.cn.", dns.TypeA, "udp", "114.114.114.114:53"},
		{"xiaomi.com.", dns.TypeA, "udp", "114.114.114.114:53"},
		{"youtube.com.", dns.TypeA, "udp", "8.8.8.8:53"},
		{"twitter.com.", dns.TypeA, "tcp", "8.8.8.8:53"},
	}

	for _, tt := range tests {
		q := dns.Question{
			Name:   tt.domain,
			Qtype:  tt.qtype,
			Qclass: dns.ClassINET,
		}

		want, _ := naiveResolve(q, true, tt.net, tt.expectedUpstream)
		got, err := naiveResolve(q, true, tt.net, "127.0.0.1:52345")

		if err != nil {
			t.Error(err)
		}

		if want == nil {
			t.Errorf("want is nil")
			continue
		}

		if got == nil {
			t.Errorf("got is nil")
			continue
		}

		if len(want.Answer) != len(got.Answer) || len(want.Question) != len(got.Question) || len(want.Extra) != len(got.Extra) {
			t.Errorf("got different resolve results from expectedUpstream and freedns")
		}
	}

	shut <- true
}
