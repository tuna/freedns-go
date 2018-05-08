package main

import (
	"log"
	"os"

	"net/http"
	_ "net/http/pprof"

	"github.com/Chenyao2333/freedns-go/freedns"
)

func main() {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	s, err := freedns.NewServer(freedns.Config{
		FastDNS:   "10.56.1.1:53",
		CleanDNS:  "8.8.8.8:53",
		Listen:    "127.0.0.1:53",
		CacheSize: 1024 * 2,
	})
	if err != nil {
		log.Fatalln(err)
		os.Exit(-1)
	}

	log.Fatalln(s.Run())
	os.Exit(-1)
}
