package main

import (
	"log"
	"os"

	"github.com/Chenyao2333/freedns-go/freedns"
)

func main() {
	s, err := freedns.NewServer(freedns.Config{
		FastDNS:  "114.114.114.114",
		CleanDNS: "8.8.8.8",
	})
	if err != nil {
		log.Fatal(err)
		os.Exit(-1)
	}

	s.Run()
}
