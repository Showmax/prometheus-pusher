package main

import (
	"strings"
)

func splitMetrics(data []byte, len int) (*sm, []byte) {
	s := &sm{
		lines: strings.Split(string(data), "\n"),
		size:  len,
	}
	return s, s.Read()
}

type sm struct {
	lines  []string
	offset int
	size   int
}

func (d *sm) Read() []byte {
	tosend := []string{}
	left := d.size
	for i := d.offset; left > 0 && i < len(d.lines); i++ {
		line := d.lines[i]
		if strings.HasPrefix(line, "#") {
			d.offset += 1
			tosend = append(tosend, line)
			continue
		}
		left -= 1
		d.offset += 1
		tosend = append(tosend, d.lines[i])
	}
	if len(tosend) > 0 {
		return []byte(strings.Join(tosend, "\n") + "\n")
	}
	return nil

}
