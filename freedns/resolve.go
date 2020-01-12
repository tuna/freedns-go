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
	fastUpstream  string
	cleanUpstream string

	// cnDomains caches if a domain belongs to China.
	cnDomains *goc.Cache
}

// resovle returns the response and which upstream is used
func (resolver *spoofingProofResolver) resolve(req *dns.Msg, net string) (*dns.Msg, string) {
	type result struct {
		res *dns.Msg
		err error
	}
	fastCh := make(chan result, 4)
	cleanCh := make(chan result, 4)

	Q := func(ch chan result, upstream string) {
		res, err := naiveResolve(req, upstream, net)
		ch <- result{res, err}
	}

	go Q(cleanCh, resolver.cleanUpstream)
	go Q(fastCh, resolver.fastUpstream)

	// send timeout results
	go func() {
		time.Sleep(2 * time.Second)
		fastCh <- result{nil, Error("timeout")}
		cleanCh <- result{nil, Error("timeout")}
	}()

	// 1. if we can distinguish if it is china domain, we directly uses the right upstream
	isCN, ok := resolver.cnDomains.Get(req.Question[0].Name)
	if ok {
		if isCN.(bool) {
			r := <-fastCh
			return r.res, resolver.fastUpstream
		}
		r := <-cleanCh
		return r.res, resolver.cleanUpstream
	}

	// 2. try to resolve by fast dns. if it contains A record which means we can decide if this is a china domain
	r := <-fastCh
	if r.res != nil && r.res.Rcode == dns.RcodeSuccess && containsA(r.res) {
		if containsChinaip(r.res) {
			resolver.cnDomains.Set(req.Question[0].Name, true)
			return r.res, resolver.fastUpstream
		}
		resolver.cnDomains.Set(req.Question[0].Name, false)
	}

	// 3. the domain may not belong to China, use the clean upstream
	r = <-cleanCh
	if r.res == nil {
		r.res = &dns.Msg{}
		r.res.SetRcode(req, dns.RcodeServerFailure)
	}
	return r.res, resolver.cleanUpstream
}

func naiveResolve(req *dns.Msg, upstream string, net string) (*dns.Msg, error) {
	r := req.Copy()
	r.Id = dns.Id()

	c := &dns.Client{Net: net}

	res, _, err := c.Exchange(r, upstream)

	if err != nil {
		log.WithFields(logrus.Fields{
			"op":       "naive_resolve",
			"upstream": upstream,
			"domain":   req.Question[0].Name,
		}).Error(err)
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
