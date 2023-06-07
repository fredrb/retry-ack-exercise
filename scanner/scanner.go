package scanner

import (
	"context"
	"log"
	"time"
)

type Scanner struct {
	peer          Peer
	osScanner     OSInterface
	peerMessages  chan []string
	runScanSignal <-chan time.Time

	lastSent []string
}

// OSInterface interacts with teh Operating System and runs a full scan.
// The implementation will vary on the OS.
type OSInterface interface {
	GetScanResults() ([]string, error)
}

type Peer interface {
	// ScanReceivedAckC returns a channel that can be read by the scanner to know when
	// the peer has successfully processed a scan.
	ScanReceivedAckC() chan struct{}

	// StopSignal will be emitted by the peer when shutting down. Scanner should stop
	// when this happens and don't write into the stream anymore.
	StopSignal() chan struct{}

	// SendScan will write the scan into an unreliable interface. The only way to know if
	// the scan was received by the Peer is by reading from the ACK channel.
	SendScan([]string)
}

func NewScanner(peer Peer, os OSInterface, runScan <-chan time.Time) Scanner {
	return Scanner{
		peer:          peer,
		osScanner:     os,
		runScanSignal: runScan,
		peerMessages:  make(chan []string),
	}
}

func (s *Scanner) manageScanLoop(ctx context.Context) {
	for {
		select {
		case <-s.runScanSignal:
			result, err := s.osScanner.GetScanResults()
			if err != nil {
				log.Printf("Failed to get scan results")
				break
			}
			s.peerMessages <- result
		case <-ctx.Done():
			return
		}
	}
}

func (s *Scanner) manageStream(ctx context.Context) {
	for {
		select {
		case <-s.peer.ScanReceivedAckC():
			// TODO: process ACK from stream
		case m, more := <-s.peerMessages:
			if !more {
				return
			}
			s.lastSent = m
			s.peer.SendScan(m)
		case <-ctx.Done():
			if len(s.peerMessages) != 0 {
				log.Printf("Warning: stopping stream with %d messages in the queue", len(s.peerMessages))
			}
			return
		}
	}
}

func (s *Scanner) Start() {
	log.Printf("Starting Scanner component")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		ticker := time.NewTicker(time.Second)
		for {
			select {
			case <-ticker.C:
				s.peerMessages <- s.lastSent
			}
		}
	}()

	go s.manageScanLoop(ctx)
	go s.manageStream(ctx)

	<-s.peer.StopSignal()
}
