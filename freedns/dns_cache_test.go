package freedns

import (
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
)

func TestAll(t *testing.T) {
	a_query := &dns.Msg{
		MsgHdr: dns.MsgHdr{
			RecursionDesired: true,
		},
		Question: []dns.Question{dns.Question{
			Name:   "example.com",
			Qtype:  dns.TypeA,
			Qclass: dns.ClassANY,
		}},
		Answer: []dns.RR{
			&dns.A{
				Hdr: dns.RR_Header{
					Name: "example.com",
					Ttl:  5,
				},
				A: net.IPv4(127, 0, 0, 1),
			},
		},
	}

	c := new_dns_cache(10)
	c.set(a_query, "udp")
	time.Sleep(1 * time.Second)
	res, upd := c.lookup(a_query.Question[0], a_query.RecursionDesired, "udp")
	if res.Answer[0].(*dns.A).Hdr.Name != a_query.Answer[0].(*dns.A).Hdr.Name {
		t.Errorf("lookup returns wrong result!")
	}
	if upd || res.Answer[0].(*dns.A).Hdr.Ttl <= 3 {
		t.Errorf("the ttl should be 4 and do not need to update")
	}
	time.Sleep(1 * time.Second)
	res, upd = c.lookup(a_query.Question[0], a_query.RecursionDesired, "udp")
	if !upd || res.Answer[0].(*dns.A).Hdr.Ttl > 3 {
		t.Errorf("the tll should be no more than 3 and need to update")
	}
}
