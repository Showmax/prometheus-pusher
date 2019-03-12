package main

import (
	"bytes"
	"fmt"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

// metrics scanner
//
type metrics struct {
	cNl    bool        // semaphore for new line capture
	cName  bool        // semaphore for metric name capture
	cData  bool        // semaphore for metric data capture
	cCmt   bool        // semaphore for comment capture
	cBrace int         // counter for curly brace capture
	dBrd   [][3]uint64 // data borders map
	dCmt   [][2]uint64 // comment borders map
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
// It works by reading a sections of scanned data governed by `dBrd`
// (which says where name and metric borders are) then trims leading and
// ending whitespaces and splits the data into fields by the remaining whitespaces
// If there are less than 3 fields, timestamp is added.
//
func newMetric(m *metrics, idx int, rm *routeMap, ts *[]byte, cfg *pusherConfig) *metric {
	mf := bytes.Fields(m.bytes[m.dBrd[idx][0]:m.dBrd[idx][2]])

	var buffer bytes.Buffer
	// Add labels from environment if configured
	req, err := http.NewRequest("GET", "http://localhost", nil)
	req.Header.Set("Content-Type", "text/plain")
	var (
		allSamples = make(model.Samples, 0, 1)
		decSamples = make(model.Vector, 0, 1)
	)
	fullMetricLine := bytes.Join(mf, []byte{' '})
	fullMetricLine = bytes.Join([][]byte{fullMetricLine, []byte("\n")}, []byte{})
	sdec := expfmt.SampleDecoder{
		Dec:  expfmt.NewDecoder(ioutil.NopCloser(bytes.NewReader(fullMetricLine)), expfmt.ResponseFormat(req.Header)),
		Opts: &expfmt.DecodeOptions{},
	}

	for {
		if err = sdec.Decode(&decSamples); err != nil {
			// SampleDecoder returns EOF to throw the end of metrics
			//  https://github.com/prometheus/common/blob/master/expfmt/decode.go#L132
			if err == io.EOF {
				err = nil
				break
			}
			logger.Warnf("Cannot parse metric %s due to %s", string(mf[0]), err)
			// In case something goes wrong let's fallback to original solution
			return &metric{
				dsts:  rm.route(m.bytes[m.dBrd[idx][0]:m.dBrd[idx][1]]),
				bytes: bytes.Join(append(mf, *ts), []byte{' '}),
			}
			break
		}
		allSamples = append(allSamples, decSamples...)
		// decSamples = decSamples[:0]
	}

	for _, sample := range allSamples {
		if len(cfg.envLabels) > 0 {
			for labelName, labelValue := range cfg.envLabels {
				sample.Metric[model.LabelName(labelName)] = model.LabelValue(labelValue)
			}
		}
		metric := fmt.Sprintf("%s %s %s", sample.Metric, sample.Value, *ts)
		buffer.WriteString(metric)
	}

	return &metric{
		dsts:  rm.route(m.bytes[m.dBrd[idx][0]:m.dBrd[idx][1]]),
		bytes: buffer.Bytes(),
	}
}

// metrics scanner constructor
// it also automatically runs the scanner
//
func newMetrics(bytes []byte, cfg *pusherConfig) *metrics {
	m := &metrics{
		bytes:  bytes,
		cNl:    true,
		cName:  false,
		cData:  false,
		cCmt:   false,
		cBrace: 0,
		dBrd:   make([][3]uint64, 0),
		dCmt:   make([][2]uint64, 0),
	}
	return m.scan(cfg)
}

func newComment(m *metrics, idx int) []byte {
	return append(m.bytes[m.dCmt[idx][0]:m.dCmt[idx][1]], '\n')
}

// scans bytes byte by byte and marks indices
// of metric name and bytes borders of self
//
func (m *metrics) scan(cfg *pusherConfig) *metrics {
	for idx, char := range m.bytes {
		switch {
		case m.isValidNameChar(idx) && m.isOnNewLine(): // [a-zA-Z0-9_] character on new line
			m.startMetricCapture(idx)
		case char == 35 && m.isOnNewLine(): // comment char on new line
			m.unflagNewline()
			m.startCommentCapture(idx)
		case char == 10: // newline character
			m.flagNewline()
			if m.isCapturingMetricData() {
				m.stopMetricDataCapture(idx)
			}
			if m.isCapturingComment() {
				m.stopCommentCapture(idx)
			}
		case char == 9 || char == 32: // tab or space characters while capturing a name
			if m.isInBraces() {
				continue
			}
			if m.isCapturingMetricName() {
				m.stopMetricNameCapture(idx)
			}
			if m.isOnNewLine() { // hack for metrics with spaces before name
				m.flagNewline()
			}
		case char == 123: // open curly brace outside comment
			if !m.isCapturingComment() { // outside comment
				m.openBrace()
			} else if m.isCapturingMetricName() { // after metric name
				m.stopMetricNameCapture(idx)
			}
		case char == 125 && !m.isCapturingComment(): // close curly brace outside comment
			m.closeBrace()
		default: // any other character
			if m.isOnNewLine() {
				m.unflagNewline()
			}
		}
		if m.isLastChar(idx) {
			if m.isCapturingMetricData() { // end of data
				m.stopMetricDataCapture(idx)
			}
			if m.isCapturingComment() {
				m.stopCommentCapture(idx)
			}
		}
	}
	return m
}

// Inverse-multiplexes bytes into buckets by their destination
// and adds missing timestamps.
//
// Also prepends each destination bucket with all the comment
// lines from input data.
//
func (m *metrics) imux(rm *routeMap, cfg *pusherConfig) map[string][]byte {
	// init
	r := make(map[string][]byte)
	ch := make(chan *metric, len(m.dBrd))
	ts := []byte(strconv.Itoa(int(time.Now().UnixNano() / int64(time.Millisecond))))
	cmts := make([]byte, 0)

	// map data
	for i := range m.dBrd {
		ch <- newMetric(m, i, rm, &ts, cfg)
	}
	close(ch)

	// concat all comments
	for c := range m.dCmt {
		cmts = append(cmts, newComment(m, c)...)
	}

	// reduce []byte and prepend with comments
	for metric := range ch {
		for _, dst := range metric.dsts {
			if len(r[dst]) == 0 {
				r[dst] = append(r[dst], cmts...)
			}
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

func (m *metrics) isCapturingComment() bool {
	return m.cCmt
}

func (m *metrics) isCapturingMetricName() bool {
	return m.cName
}

func (m *metrics) isCapturingMetricData() bool {
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

func (m *metrics) startMetricCapture(idx int) {
	m.dBrd = append(m.dBrd, [3]uint64{uint64(idx), uint64(idx), uint64(idx)})
	m.unflagNewline()
	m.cName = true
	m.cData = true
}

func (m *metrics) stopMetricNameCapture(idx int) {
	m.dBrd[len(m.dBrd)-1][1] = uint64(idx)
	m.cName = false
}

func (m *metrics) stopMetricDataCapture(idx int) {
	m.dBrd[len(m.dBrd)-1][2] = uint64(idx)
	m.cData = false
}

func (m *metrics) startCommentCapture(idx int) {
	m.dCmt = append(m.dCmt, [2]uint64{uint64(idx), uint64(idx)})
	m.cCmt = true
}

func (m *metrics) stopCommentCapture(idx int) {
	m.dCmt[len(m.dCmt)-1][1] = uint64(idx)
	m.cCmt = false
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
