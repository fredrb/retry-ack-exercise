package scanner

import (
	"context"
	"log"
	"time"
)

type Scanner struct {
	peer         Peer
	osScanner    OSInterface
	peerMessages chan []string
	ticker       time.Duration
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

func NewScanner(peer Peer, os OSInterface, timer time.Duration) Scanner {
	return Scanner{
		peer:         peer,
		osScanner:    os,
		ticker:       timer,
		peerMessages: make(chan []string, 100),
	}
}

func (s *Scanner) manageScanLoop(ctx context.Context) {
	scanTicker := time.NewTicker(s.ticker)
	for {
		select {
		case <-scanTicker.C:
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

	go s.manageScanLoop(ctx)
	go s.manageStream(ctx)

	<-s.peer.StopSignal()
}
