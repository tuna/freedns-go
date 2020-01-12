package freedns

import (
	"time"

	goc "github.com/louchenyao/golang-cache"
	"github.com/miekg/dns"
)

type cacheEntry struct {
	putin time.Time
	reply *dns.Msg
}

type dnsCache struct {
	backend *goc.Cache
}

func newDNSCache(maxCap int) *dnsCache {
	c, _ := goc.NewCache("lru", maxCap)
	return &dnsCache{
		backend: c,
	}
}

func (c *dnsCache) set(res *dns.Msg, net string) {
	key := requestToString(res.Question[0], res.RecursionDesired, net)

	c.backend.Set(key, cacheEntry{
		putin: time.Now(),
		reply: res.Copy(),
	})
}

func (c *dnsCache) lookup(q dns.Question, recursion bool, net string) (*dns.Msg, bool) {
	key := requestToString(q, recursion, net)
	ci, ok := c.backend.Get(key)
	if ok {
		entry := ci.(cacheEntry)
		res := entry.reply.Copy()
		delta := time.Now().Sub(entry.putin).Seconds()
		needUpdate := subTTL(res, int(delta))

		return res, needUpdate
	}
	return nil, true
}

// requestToString generates a string that uniquely identifies the request.
func requestToString(q dns.Question, recursion bool, net string) string {
	s := q.Name + "_" + dns.TypeToString[q.Qtype] + "_" + dns.ClassToString[q.Qclass]
	if recursion {
		s += "_1"
	} else {
		s += "_0"
	}
	s += "_" + net
	return s
}

// subTTL substracts the ttl of `res` by delta in place,
// and returns true if it will be expired in 3 seconds.
func subTTL(res *dns.Msg, delta int) bool {
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
