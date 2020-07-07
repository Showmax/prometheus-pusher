package main

import (
	"testing"
)

func TestSplitMetrics(t *testing.T) {

	input := `#
1
2
#
3
#
1
#
2
#
3
1
2
3`
	expected := []string{`#
1
2
#
3
`,
		`#
1
#
2
#
3
`,
		`1
2
3
`}
	s, part := splitMetrics([]byte(input), 3)
	for i := 0; i < 3; i++ {
		if string(part) != expected[i] {
			t.Fatalf("[%d] got  ==%s==, expected ==%s==", i, string(part), expected[i])
		}
		part = s.Read()
	}
	part = s.Read()
	if part != nil {
		t.Fatalf("expected nil")
	}
}
