package freedns

import (
	"time"

	goc "github.com/louchenyao/golang-cache"
	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"
	"github.com/tuna/freedns-go/chinaip"
)

// spoofing_proof_resolver can resolve the DNS request with 100% confidence.
type spoofing_proof_resolver struct {
	fast_upstream  string
	clean_upstream string

	// cn_domains caches if a domain belongs to China.
	cn_domains *goc.Cache
}

// resovle returns the response and which upstream is used
func (this *spoofing_proof_resolver) resolve(req *dns.Msg, net string) (*dns.Msg, string) {
	type result struct {
		res *dns.Msg
		err error
	}
	fast_ch := make(chan result, 4)
	clean_ch := make(chan result, 4)

	Q := func(ch chan result, upstream string) {
		res, err := naive_resolve(req, upstream, net)
		ch <- result{res, err}
	}

	go Q(clean_ch, this.clean_upstream)
	go Q(fast_ch, this.fast_upstream)

	// send timeout results
	go func() {
		time.Sleep(2 * time.Second)
		fast_ch <- result{nil, Error("timeout")}
		clean_ch <- result{nil, Error("timeout")}
	}()

	// 1. if we can distinguish if it is china domain, we directly uses the right upstream
	is_cn, ok := this.cn_domains.Get(req.Question[0].Name)
	if ok {
		if is_cn.(bool) {
			r := <-fast_ch
			return r.res, this.fast_upstream
		} else {
			r := <-clean_ch
			return r.res, this.clean_upstream
		}
	}

	// 2. try to resolve by fast dns. if it contains A record which means we can decide if this is a china domain
	r := <-fast_ch
	if r.res != nil && r.res.Rcode == dns.RcodeSuccess && containsA(r.res) {
		if containsChinaip(r.res) {
			this.cn_domains.Set(req.Question[0].Name, true)
			return r.res, this.fast_upstream
		} else {
			this.cn_domains.Set(req.Question[0].Name, false)
		}
	}

	// 3. the domain may not belong to China, use the clean upstream
	r = <-clean_ch
	if r.res == nil {
		r.res = &dns.Msg{}
		r.res.SetRcode(req, dns.RcodeServerFailure)
	}
	return r.res, this.clean_upstream
}

func naive_resolve(req *dns.Msg, upstream string, net string) (*dns.Msg, error) {
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
