package freedns

import (
	"time"

	goc "github.com/louchenyao/golang-cache"
	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"
	"github.com/tuna/freedns-go/chinaip"
)

// spoofingProofResolver can resolve the DNS request with 100% confidence.
type spoofingProofResolver struct {
	fastUpstreamProvider  upstreamProvider
	cleanUpstreamProvider upstreamProvider

	// cnDomains caches if a domain belongs to China.
	cnDomains *goc.Cache
}

func newSpoofingProofResolver(fastUpstreamProvider upstreamProvider, cleanUpstreamProvider upstreamProvider, cacheCap int) *spoofingProofResolver {
	c, _ := goc.NewCache("lru", cacheCap)
	return &spoofingProofResolver{
		fastUpstreamProvider:  fastUpstreamProvider,
		cleanUpstreamProvider: cleanUpstreamProvider,
		cnDomains:             c,
	}
}

// resovle returns the response and which upstream is used
func (resolver *spoofingProofResolver) resolve(q dns.Question, recursion bool, net string) (*dns.Msg, string) {
	type result struct {
		res *dns.Msg
		err error
	}
	fastCh := make(chan result, 4)
	cleanCh := make(chan result, 4)

	fail := &dns.Msg{
		MsgHdr: dns.MsgHdr{
			Rcode: dns.RcodeServerFailure,
		},
	}

	Q := func(ch chan result, upstream string) {
		res, err := naiveResolve(q, recursion, net, upstream)
		if res == nil {
			res = fail
		}
		ch <- result{res, err}
	}

	cleanUpstream := resolver.cleanUpstreamProvider.GetUpstream()
	fastUpstream := resolver.fastUpstreamProvider.GetUpstream()

	go Q(cleanCh, cleanUpstream)
	go Q(fastCh, fastUpstream)

	// send timeout results
	go func() {
		time.Sleep(1900 * time.Millisecond)
		fastCh <- result{fail, Error("timeout")}
		cleanCh <- result{fail, Error("timeout")}
	}()

	// 1. if we can distinguish if it is a china domain, we directly uses the right upstream
	isCN, ok := resolver.cnDomains.Get(q.Name)
	if ok {
		if isCN.(bool) {
			r := <-fastCh
			// The fast upstream returns the success result
			if r.res != nil && r.res.Rcode == dns.RcodeSuccess {
				// recheck if it is a china domain, and update the cache
				// we do this recheck in case that the GFW spoofs the domain and returns an IP in China
				if containsA(r.res) && !containsChinaip(r.res) {
					resolver.cnDomains.Set(q.Name, false)
				} else {
					return r.res, fastUpstream
				}
			}
		}
		r := <-cleanCh
		return r.res, cleanUpstream
	}

	// 2. try to resolve by fast dns. if it contains A record which means we can decide if this is a china domain
	r := <-fastCh
	if r.res != nil && r.res.Rcode == dns.RcodeSuccess && containsA(r.res) {
		if containsChinaip(r.res) {
			resolver.cnDomains.Set(q.Name, true)
			return r.res, fastUpstream
		}
		resolver.cnDomains.Set(q.Name, false)
	}

	// 3. the domain may not belong to China, use the clean upstream
	r = <-cleanCh
	return r.res, cleanUpstream
}

func naiveResolve(q dns.Question, recursion bool, net string, upstream string) (*dns.Msg, error) {
	r := &dns.Msg{
		MsgHdr: dns.MsgHdr{
			Id:               dns.Id(),
			RecursionDesired: recursion,
		},
		Question: []dns.Question{q},
	}
	c := &dns.Client{Net: net}

	res, _, err := c.Exchange(r, upstream)

	if err != nil {
		log.WithFields(logrus.Fields{
			"op":       "naive_resolve",
			"upstream": upstream,
			"domain":   q.Name,
		}).Error(err)
		// In case the Rcode is initialized as RcodeSuccess but the error occurs.
		// Without this, the wrong result may be cached and returned.
		if res != nil && res.Rcode == dns.RcodeSuccess {
			res = nil
		}
	}

	return res, err
}

func containsA(res *dns.Msg) bool {
	var rrs []dns.RR

	rrs = append(rrs, res.Answer...)
	rrs = append(rrs, res.Ns...)
	rrs = append(rrs, res.Extra...)

	for i := 0; i < len(rrs); i++ {
		_, ok := rrs[i].(*dns.A)
		if ok {
			return true
		}
	}
	return false
}

// containChinaIP check if the resoponse contains IP belonging to China.
func containsChinaip(res *dns.Msg) bool {
	var rrs []dns.RR

	rrs = append(rrs, res.Answer...)
	rrs = append(rrs, res.Ns...)
	rrs = append(rrs, res.Extra...)

	for i := 0; i < len(rrs); i++ {
		rr, ok := rrs[i].(*dns.A)
		if ok {
			ip := rr.A.String()
			if chinaip.IsChinaIP(ip) {
				return true
			}
		}
	}
	return false
}
