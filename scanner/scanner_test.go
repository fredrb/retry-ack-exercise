package scanner

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type spyPeer struct {
	ackC  chan struct{}
	stopC chan struct{}

	sentScans chan []string
}

func (p *spyPeer) ScanReceivedAckC() chan struct{} {
	return p.ackC
}

func (p *spyPeer) SendScan(s []string) {
	p.sentScans <- s
}

func (p *spyPeer) StopSignal() chan struct{} {
	return p.stopC
}

func (p *spyPeer) waitForMessages(numMessage int, timeout time.Duration) ([][]string, error) {
	receivedMessages := make([][]string, numMessage)
	for i := 0; i < numMessage; i++ {
		select {
		case msg, more := <-p.sentScans:
			if !more {
				return nil, errors.New("should not have closed")
			}
			receivedMessages[i] = msg
		case <-time.After(timeout):
			return nil, errors.New("timeout reached waiting for scan result")
		}
	}
	return receivedMessages, nil
}

type fakeOs struct {
	orderedResults []string
	timesCalled    int
}

func (o *fakeOs) GetScanResults() ([]string, error) {
	var index int
	if o.timesCalled >= len(o.orderedResults) {
		index = len(o.orderedResults) - 1
	} else {
		index = o.timesCalled
	}
	o.timesCalled++
	return []string{o.orderedResults[index]}, nil
}

func Test_ScannerWithoutAck(t *testing.T) {
	os := fakeOs{
		orderedResults: []string{"c1", "c2", "c3"},
	}
	spy := spyPeer{
		ackC:      make(chan struct{}, 1),
		stopC:     make(chan struct{}, 1),
		sentScans: make(chan []string, 100),
	}

	scanSignal := make(chan time.Time)
	s := NewScanner(&spy, &os, scanSignal)
	go s.Start()

	scanSignal <- time.Now()
	_, err := spy.waitForMessages(1, 3*time.Second)
	require.NoError(t, err, "Didn't receive original message")
	_, err = spy.waitForMessages(1, 3*time.Second)
	require.NoError(t, err, "Didn't receive retry message")
}

func Test_ScannerWithAck(t *testing.T) {
	os := fakeOs{
		orderedResults: []string{"c1", "c2", "c3"},
	}
	spy := spyPeer{
		ackC:      make(chan struct{}, 1),
		stopC:     make(chan struct{}, 1),
		sentScans: make(chan []string, 100),
	}

	scanSignal := make(chan time.Time)
	s := NewScanner(&spy, &os, scanSignal)
	go s.Start()

	scanSignal <- time.Now()
	_, err := spy.waitForMessages(1, 3*time.Second)
	require.NoError(t, err, "Didn't receive original message")

	spy.ScanReceivedAckC() <- struct{}{}
	// No messages written

	select {
	case <-spy.sentScans:
		t.Fatalf("should not receive scan")
	case <-time.After(3 * time.Second):
		break
	}
}

func Test_RunScanner(t *testing.T) {
	os := fakeOs{
		orderedResults: []string{"c1", "c2", "c3"},
	}
	spy := spyPeer{
		ackC:      make(chan struct{}, 1),
		stopC:     make(chan struct{}, 1),
		sentScans: make(chan []string, 100),
	}

	scanSignal := make(chan time.Time)
	s := NewScanner(&spy, &os, scanSignal)
	go s.Start()

	for i := 0; i < 5; i++ {
		scanSignal <- time.Now()
	}
	time.Sleep(time.Millisecond)

	receivedMessage, err := spy.waitForMessages(4, 10*time.Second)
	require.NoError(t, err)

	require.Len(t, receivedMessage, 4)
	assert.Equal(t, receivedMessage[0][0], "c1")
	assert.Equal(t, receivedMessage[1][0], "c2")
	assert.Equal(t, receivedMessage[2][0], "c3")
	assert.Equal(t, receivedMessage[3][0], "c3")
}
