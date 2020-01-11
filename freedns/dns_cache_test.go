package freedns

import (
	"testing"

	"github.com/miekg/dns"
)

func TestAll(t *testing.T) {
	c = new_dns_cache(10)
	dns.Msg{
		Question: {Msg.Question},
	}
	c.set()
}
