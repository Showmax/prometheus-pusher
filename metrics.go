package main

import (
	"bytes"
	"strconv"
	"sync"
	"time"
)

// metrics scanner
//
type metrics struct {
	cNl    bool        // semaphore for new line capture
	cName  bool        // semaphore for metric name capture
	cData  bool        // semaphore for metric data capture
	cBrace int         // counter for curly brace capture
	brd    [][3]uint64 // data borders map
	bytes  []byte      // metrics data payload
}

// metric bytes chunk with its destination
// used to send bytes via channel within `mux()`
// method
//
type metric struct {
	dsts  []string // routing destination
	bytes []byte   // metric data payload
}

// create metric from metrics' []byte payload
//
// It works by reading a sections of scanned data governed by `brd`
// (which says where name and metric borders are) then trims leading and
// ending whitespaces and splits the data into fields by the remaining whitespaces
// If there are less than 3 fields, timestamp is added.
//
func newMetric(m *metrics, idx int, ch chan *metric, rm *routeMap, ts *[]byte, wg *sync.WaitGroup) {
	defer wg.Done()

	mf := bytes.Fields(m.bytes[m.brd[idx][1]:m.brd[idx][2]])
	// logger.Error(string(mf[0]))
	if len(mf) < 2 { // no timestamp
		mf = append(mf, *ts)
	}
	mf = append([][]byte{m.bytes[m.brd[idx][0]:m.brd[idx][1]]}, mf...)

	ch <- &metric{
		dsts:  rm.route(m.bytes[m.brd[idx][0]:m.brd[idx][1]]),
		bytes: bytes.Join(mf, []byte{' '}),
	}
}

// metrics scanner constructor
// it also automatically runs the scanner
//
func newMetrics(bytes []byte) *metrics {
	m := &metrics{
		bytes:  bytes,
		cNl:    true,
		cName:  false,
		cData:  false,
		cBrace: 0,
		brd:    make([][3]uint64, 0),
	}
	return m.scan()
}

// scans bytes byte by byte and marks indices
// of metric name and bytes borders of self
//
func (m *metrics) scan() *metrics {
	for idx, char := range m.bytes {
		switch {
		case m.isValidNameChar(idx) && m.isOnNewLine(): // [a-zA-Z0-9_] character on new line
			m.startCapture(idx)
		case char == 10: // newline character
			m.flagNewline()
			if m.isCapturingData() {
				m.stopDataCapture(idx)
			}
		case char == 9 || char == 32: // tab or space characters while capturing a name
			if m.isInBraces() {
				continue
			}
			if m.isCapturingName() {
				m.stopNameCapture(idx)
			}
			if m.isOnNewLine() { // hack for metrics with spaces before name
				m.flagNewline()
			}
		case char == 123:
			m.openBrace()
		case char == 125:
			m.closeBrace()
		case char == 35 && m.isOnNewLine(): // comment char on new line
			m.unflagNewline()
		case char == 123: // open curly brace
			if m.isCapturingName() {
				m.stopNameCapture(idx)
			}
		default: // any other character
			if m.isOnNewLine() {
				m.unflagNewline()
			}
		}
		if m.isLastChar(idx) && m.isCapturingData() { // end of data
			m.stopDataCapture(idx)
		}
	}
	return m
}

// inverse-multiplexes bytes into buckets by their destination
// and adds missing timestamps
//
func (m *metrics) imux(rm *routeMap) map[string][]byte {
	// init
	r := make(map[string][]byte)
	ch := make(chan *metric, len(m.brd))
	wg := &sync.WaitGroup{}
	ts := []byte(strconv.Itoa(int(time.Now().UnixNano() / int64(time.Millisecond))))

	// map
	wg.Add(len(m.brd))
	for i := range m.brd {
		go newMetric(m, i, ch, rm, &ts, wg)
	}

	// wait for mappers to finish
	wg.Wait()
	close(ch)

	// reduce
	for metric := range ch {
		for _, dst := range metric.dsts {
			r[dst] = append(r[dst], metric.bytes...)
			r[dst] = append(r[dst], byte('\n'))
		}
	}

	return r
}

/*
 * HELPER METHODS
 *
 * These serve mainly to make the decision tree sane
 * enough to be comprehended easily. Their names are
 * self-explanatory.
 */

func (m *metrics) isCapturingName() bool {
	return m.cName
}

func (m *metrics) isCapturingData() bool {
	return m.cData
}

func (m *metrics) isInBraces() bool {
	return m.cBrace != 0
}

func (m *metrics) isLastChar(idx int) bool {
	return len(m.bytes) == idx+1
}

func (m *metrics) isOnNewLine() bool {
	return m.cNl
}

func (m *metrics) isValidNameChar(idx int) bool {
	switch {
	case m.bytes[idx] >= 97 && m.bytes[idx] <= 122: // a-z
		return true
	case m.bytes[idx] == 95: // _
		return true
	case m.bytes[idx] >= 65 && m.bytes[idx] <= 90: // A-Z
		return true
	case m.bytes[idx] >= 48 && m.bytes[idx] <= 57: // 0-9
		return true
	default:
		return false
	}
}

func (m *metrics) startCapture(idx int) {
	m.brd = append(m.brd, [3]uint64{uint64(idx), uint64(idx), uint64(idx)})
	m.unflagNewline()
	m.cName = true
	m.cData = true
}

func (m *metrics) stopNameCapture(idx int) {
	m.brd[len(m.brd)-1][1] = uint64(idx)
	m.cName = false
}

func (m *metrics) stopDataCapture(idx int) {
	m.brd[len(m.brd)-1][2] = uint64(idx)
	m.cData = false
}

func (m *metrics) flagNewline() {
	m.cNl = true
}

func (m *metrics) unflagNewline() {
	m.cNl = false
}

func (m *metrics) openBrace() {
	m.cBrace++
}

func (m *metrics) closeBrace() {
	m.cBrace--
}
