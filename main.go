package main

import (
	"github.com/fredrb/retry-ack-exercise/scanner"
	"log"
	"time"
)

type fakePeer struct {
	ackC  chan struct{}
	stopC chan struct{}
}

func (p *fakePeer) ScanReceivedAckC() chan struct{} {
	return p.ackC
}

func (p *fakePeer) SendScan(s []string) {
	log.Printf("scan received %+v\n", s)
}

func (p *fakePeer) StopSignal() chan struct{} {
	return p.stopC
}

type fakeOs struct{}

func (o *fakeOs) GetScanResults() ([]string, error) {
	return []string{"c1"}, nil
}

func main() {

	peer := fakePeer{
		ackC:  make(chan struct{}, 1),
		stopC: make(chan struct{}, 1),
	}
	os := fakeOs{}

	s := scanner.NewScanner(&peer, &os, time.Second)
	go s.Start()

	time.Sleep(5 * time.Second)
	peer.StopSignal() <- struct{}{}
	time.Sleep(1 * time.Second)

}
