/*
Copyright 2014 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// A small utility to just serve debug info on TCP and/or UDP.

package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	doTCP  bool
	doUDP  bool
	doHTTP bool
	port   int
)

func init() {
	flag.BoolVar(&doTCP, "tcp", false, "serve raw over TCP")
	flag.BoolVar(&doUDP, "udp", false, "serve raw over UDP")
	flag.BoolVar(&doHTTP, "http", false, "serve HTTP")
	flag.IntVar(&port, "port", 9376, "port number")
}

func main() {
	flag.Parse()

	if !doHTTP && !doTCP && !doUDP {
		doHTTP = true
	}
	if doHTTP && (doTCP || doUDP) {
		log.Fatalf("can't serve TCP/UDP mode and HTTP mode at the same time")
	}

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("error from os.Hostname(): %s", err)
	}

	if doTCP {
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			log.Fatalf("Listen(): %s", err)
		}
		go func() {
			log.Printf("serving TCP on port %d\n", port)
			for {
				conn, err := listener.Accept()
				if err != nil {
					log.Fatalf("Accept(): %s", err)
				}
				client := conn.RemoteAddr().String()
				log.Printf("TCP request from %s", client)
				conn.Write([]byte(makeMessage(hostname, client)))
				conn.Close()
			}
		}()
	}
	if doUDP {
		addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port))
		if err != nil {
			log.Fatalf("ResolveUDPAddr(): %s", err)
		}
		sock, err := net.ListenUDP("udp", addr)
		if err != nil {
			log.Fatalf("ListenUDP(): %s", err)
		}
		go func() {
			log.Printf("serving UDP on port %d\n", port)
			var buffer [16]byte
			for {
				_, cliAddr, err := sock.ReadFrom(buffer[0:])
				if err != nil {
					log.Fatalf("ReadFrom(): %s", err)
				}
				log.Printf("UDP request from %s", cliAddr.String())
				sock.WriteTo([]byte(makeMessage(hostname, cliAddr.String())), cliAddr)
			}
		}()
	}
	if doHTTP {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// Add this header to force to close the connection after serving the request.
			w.Header().Add("Connection", "close")

			log.Printf("HTTP request from %s", r.RemoteAddr)
			fmt.Fprintf(w, "%s", makeMessage(hostname, r.RemoteAddr))
		})
		go func() {
			log.Printf("serving HTTP on port %d\n", port)
			log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
		}()
	}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM)
	sig := <-signals
	log.Printf("shutting down after receiving signal: %s\n", sig)
	log.Printf("awaiting pod deletion.\n")
	time.Sleep(60 * time.Second)
}

func makeMessage(hostname, client string) string {
	return fmt.Sprintf("{\"server\":%q, \"client\":%q}\n", hostname, client)
}
