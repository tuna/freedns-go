package freedns

import (
	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"
)

// Config stores the configuration for the Server
type Config struct {
	FastUpstream  string
	CleanUpstream string
	Listen        string
	CacheCap      int // the maximum items can be cached
}

// Server is type of the freedns server instance
type Server struct {
	config Config

	udpServer *dns.Server
	tcpServer *dns.Server

	resolver     *spoofingProofResolver
	recordsCache *dnsCache
}

var log = logrus.New()

// Error is the freedns error type
type Error string

func (e Error) Error() string {
	return string(e)
}

// NewServer creates a new freedns server instance.
func NewServer(cfg Config) (*Server, error) {
	s := &Server{}

	if cfg.Listen == "" {
		cfg.Listen = "127.0.0.1"
	}
	var err error
	if cfg.Listen, err = normalizeDnsAddress(cfg.Listen); err != nil {
		return nil, err
	}

	var fastUpstreamProvider, cleanUpstreamProvider upstreamProvider
	fastUpstreamProvider, err = newUpstreamProvider(cfg.FastUpstream)
	if err != nil {
		return nil, err
	}
	cleanUpstreamProvider, err = newUpstreamProvider(cfg.CleanUpstream)
	if err != nil {
		return nil, err
	}

	s.config = cfg
	s.udpServer = &dns.Server{
		Addr: s.config.Listen,
		Net:  "udp",
		Handler: dns.HandlerFunc(func(w dns.ResponseWriter, req *dns.Msg) {
			s.handle(w, req, "udp")
		}),
	}

	s.tcpServer = &dns.Server{
		Addr: s.config.Listen,
		Net:  "tcp",
		Handler: dns.HandlerFunc(func(w dns.ResponseWriter, req *dns.Msg) {
			s.handle(w, req, "tcp")
		}),
	}

	s.recordsCache = newDNSCache(cfg.CacheCap)

	s.resolver = newSpoofingProofResolver(fastUpstreamProvider, cleanUpstreamProvider, cfg.CacheCap)

	return s, nil
}

// Run tcp and udp server.
func (s *Server) Run() error {
	errChan := make(chan error, 2)

	go func() {
		err := s.tcpServer.ListenAndServe()
		errChan <- err
	}()

	go func() {
		err := s.udpServer.ListenAndServe()
		errChan <- err
	}()

	select {
	case err := <-errChan:
		s.tcpServer.Shutdown()
		s.udpServer.Shutdown()
		return err
	}
}

// Shutdown shuts down the freedns server
func (s *Server) Shutdown() {
	s.tcpServer.Shutdown()
	s.udpServer.Shutdown()
}

func (s *Server) handle(w dns.ResponseWriter, req *dns.Msg, net string) {
	res := &dns.Msg{}

	if len(req.Question) < 1 {
		res.SetRcode(req, dns.RcodeBadName)
		w.WriteMsg(res)
		log.WithFields(logrus.Fields{
			"op":  "handle",
			"msg": "request without questions",
		}).Warn()
		return
	}

	res, upstream := s.lookup(req, net)
	w.WriteMsg(res)

	// logging
	l := log.WithFields(logrus.Fields{
		"op":       "handle",
		"domain":   req.Question[0].Name,
		"type":     dns.TypeToString[req.Question[0].Qtype],
		"upstream": upstream,
		"status":   dns.RcodeToString[res.Rcode],
	})
	if res.Rcode == dns.RcodeSuccess {
		l.Info()
	} else {
		l.Warn()
	}
}

// lookup queries the dns request `q` on either the local cache or upstreams,
// and returns the result and which upstream is used. It updates the local cache
// if necessary.
func (s *Server) lookup(req *dns.Msg, net string) (*dns.Msg, string) {
	// 1. lookup the cache first
	res, upd := s.recordsCache.lookup(req.Question[0], req.RecursionDesired, net)
	var upstream string

	if res != nil {
		if upd {
			go func() {
				r, u := s.resolver.resolve(req.Question[0], req.RecursionDesired, net)
				if r.Rcode == dns.RcodeSuccess {
					log.WithFields(logrus.Fields{
						"op":       "update_cache",
						"domain":   req.Question[0].Name,
						"type":     dns.TypeToString[req.Question[0].Qtype],
						"upstream": u,
					}).Info()
					s.recordsCache.set(r, net)
				}
			}()
		}
		upstream = "cache"
	} else {
		res, upstream = s.resolver.resolve(req.Question[0], req.RecursionDesired, net)
		if res.Rcode == dns.RcodeSuccess {
			log.WithFields(logrus.Fields{
				"op":       "update_cache",
				"domain":   req.Question[0].Name,
				"type":     dns.TypeToString[req.Question[0].Qtype],
				"upstream": upstream,
			}).Info()
			s.recordsCache.set(res, net)
		}
	}

	// dns.Msg.SetReply() always set the Rcode to RcodeSuccess  which we do not want
	rcode := res.Rcode
	res.SetReply(req)
	res.Rcode = rcode
	return res, upstream
}
