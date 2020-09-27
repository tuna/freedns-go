package main

import (
	"flag"
	"log"
	"os"

	_ "net/http/pprof"

	"github.com/tuna/freedns-go/freedns"
)

func main() {
	/*
		go func() {
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	*/

	var (
		fastUpstream  string
		cleanUpstream string
		listen        string
    logLevel string
	)

	flag.StringVar(&fastUpstream, "f", "114.114.114.114:53", "The fast/local DNS upstream, ip:port or resolv.conf file")
	flag.StringVar(&cleanUpstream, "c", "8.8.8.8:53", "The clean/remote DNS upstream., ip:port or resolv.conf file")
	flag.StringVar(&listen, "l", "0.0.0.0:53", "Listening address.")
	flag.StringVar(&logLevel, "log-level", "", "Set log level: info/warn/error.")

	flag.Parse()

	s, err := freedns.NewServer(freedns.Config{
		FastUpstream:  fastUpstream,
		CleanUpstream: cleanUpstream,
		Listen:        listen,
		CacheCap:      1024 * 10,
    LogLevel:      logLevel,
	})
	if err != nil {
		log.Fatalln(err)
		os.Exit(-1)
	}

	log.Fatalln(s.Run())
	os.Exit(-1)
}
