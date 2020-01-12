package freedns

import (
	"time"

	goc "github.com/louchenyao/golang-cache"
	"github.com/miekg/dns"
)

type cache_entry struct {
	putin time.Time
	reply *dns.Msg
}

type dns_cache struct {
	backend *goc.Cache
}

func new_dns_cache(max_cap int) *dns_cache {
	c, _ := goc.NewCache("lru", max_cap)
	return &dns_cache{
		backend: c,
	}
}

func (c *dns_cache) set(res *dns.Msg, net string) {
	key := request_to_string(res.Question[0], res.RecursionDesired, net)

	c.backend.Set(key, cache_entry{
		putin: time.Now(),
		reply: res.Copy(),
	})
}

func (c *dns_cache) lookup(q dns.Question, recursion bool, net string) (*dns.Msg, bool) {
	key := request_to_string(q, recursion, net)
	ci, ok := c.backend.Get(key)
	if ok {
		entry := ci.(cache_entry)
		res := entry.reply.Copy()
		delta := time.Now().Sub(entry.putin).Seconds()
		need_update := sub_ttl(res, int(delta))

		return res, need_update
	}
	return nil, true
}

// request_to_string generates a string that uniquely identifies the request.
func request_to_string(q dns.Question, recursion bool, net string) string {
	s := q.Name + "_" + dns.TypeToString[q.Qtype] + "_" + dns.ClassToString[q.Qclass]
	if recursion {
		s += "_1"
	} else {
		s += "_0"
	}
	s += "_" + net
	return s
}

// sub_ttl substracts the ttl of `res` by delta in place, 
// and returns true if it will be expired in 3 seconds.
func sub_ttl(res *dns.Msg, delta int) bool {
	needUpdate := false
	S := func(rr []dns.RR) {
		for i := 0; i < len(rr); i++ {
			newTTL := int(rr[i].Header().Ttl)
			newTTL -= delta

			if newTTL <= 3 {
				newTTL = 3
				needUpdate = true
			}

			rr[i].Header().Ttl = uint32(newTTL)
		}
	}

	S(res.Answer)
	S(res.Ns)
	S(res.Extra)

	return needUpdate
}
