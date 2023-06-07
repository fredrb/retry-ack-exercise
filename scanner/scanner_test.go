package scanner

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type spyPeer struct {
	ackC  chan struct{}
	stopC chan struct{}

	sentScans [][]string
}

func (p *spyPeer) ScanReceivedAckC() chan struct{} {
	return p.ackC
}

func (p *spyPeer) SendScan(s []string) {
	p.sentScans = append(p.sentScans, s)
}

func (p *spyPeer) StopSignal() chan struct{} {
	return p.stopC
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

func Test_RunScanner(t *testing.T) {
	os := fakeOs{
		orderedResults: []string{"c1", "c2", "c3"},
	}
	spy := spyPeer{
		ackC:      make(chan struct{}, 1),
		stopC:     make(chan struct{}, 1),
		sentScans: [][]string{},
	}

	s := NewScanner(&spy, &os, time.Millisecond)
	go s.Start()
	time.Sleep(time.Second)

	assert.Greater(t, len(spy.sentScans), 4)
	assert.Equal(t, spy.sentScans[0][0], "c1")
	assert.Equal(t, spy.sentScans[1][0], "c2")
	assert.Equal(t, spy.sentScans[2][0], "c3")
	assert.Equal(t, spy.sentScans[3][0], "c3")
}
