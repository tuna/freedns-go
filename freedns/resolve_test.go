package freedns

import (
	"testing"
	"time"

	goc "github.com/louchenyao/golang-cache"
	"github.com/miekg/dns"
)

func Test_spoofing_proof_resolver_resolve(t *testing.T) {
	c, _ := goc.NewCache("lru", 1024)
	resolver := &spoofingProofResolver{
		fastUpstream:  "114.114.114.114:53",
		cleanUpstream: "8.8.8.8:53",
		cnDomains:     c,
	}

	tests := []struct {
		domain           string
		qtype            uint16
		net              string
		expectedUpstream string
	}{
		// expect 8.8.8.8 as the upstream b/c the resolver have
		// no way to identify this is an China domain without A records
		{"ustc.edu.cn.", dns.TypeMX, "udp", "8.8.8.8:53"},
		{"ustc.edu.cn.", dns.TypeA, "udp", "114.114.114.114:53"},
		// after querying the A record of ustc.edu.cn,
		// the resolver should know this is an China domain
		{"ustc.edu.cn.", dns.TypeMX, "udp", "114.114.114.114:53"},
		{"google.com.", dns.TypeA, "udp", "8.8.8.8:53"},
		{"mi.cn.", dns.TypeA, "udp", "114.114.114.114:53"},
		{"xiaomi.com.", dns.TypeA, "udp", "114.114.114.114:53"},
		{"youtube.com.", dns.TypeA, "udp", "8.8.8.8:53"},

		// This fails just because the GitHub CI server has slow and unstable connection with the 114 server.
		//{"www.tsinghua.edu.cn.", dns.TypeA, "tcp", "114.114.114.114:53"},
		{"twitter.com.", dns.TypeA, "tcp", "8.8.8.8:53"},
	}
	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			req := &dns.Msg{
				MsgHdr: dns.MsgHdr{
					RecursionDesired: true,
				},
				Question: []dns.Question{dns.Question{
					Name:   tt.domain,
					Qtype:  tt.qtype,
					Qclass: dns.ClassINET,
				}},
			}
			start := time.Now()
			res, upstream := resolver.resolve(req, tt.net)
			end := time.Now()
			elapsed := end.Sub(start)
			if upstream != tt.expectedUpstream {
				t.Errorf("spoofing_proof_resolver.resolve() got1 = %v, want %v", upstream, tt.expectedUpstream)
			}
			if len(res.Answer) == 0 {
				t.Errorf("Expect returning at least one answer")
			}
			t.Logf("spoofing_proof_resolver.resolve() domain = %v, net = %v, elapsed = %v, record = %v", tt.domain, tt.net, elapsed, res)
		})
	}
}
