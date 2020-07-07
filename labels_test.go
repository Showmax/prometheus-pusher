package main

import (
	"reflect"
	"testing"
)

func Test_removeTimestamps(t *testing.T) {
	data := `
foo {quantile="0", pod_name="ddg-sm-2566770921-cw49q", tm_id="ddg"} 0 150610219639
bar {quantile="0.25"} 0 1506102196393
baz 1e9
# something
bae 10.3 3234242234
`
	expected := `foo {quantile="0", pod_name="ddg-sm-2566770921-cw49q", tm_id="ddg"} 0
bar {quantile="0.25"} 0
baz 1e9
# something
bae 10.3`
	results := removeTimestamps([]byte(data))
	d := string(results)
	e := string(expected)
	if e != d {
		t.Fatalf("expected: %s\n actual: %s\n", e, d)
	}
}

func Test_getEnvVarsWithPrefix(t *testing.T) {
	type testcase struct {
		prefix   string
		envvars  []string
		expected []string
	}
	testcases := []testcase{
		{
			prefix:   "log_field_",
			envvars:  []string{`log_field_foo=l1a`, `log_field_bar=l2a`, `other_baz=bam`},
			expected: []string{`foo="l1a"`, `bar="l2a"`},
		},
	}

	for idx, tc := range testcases {

		actual := getEnvVarsWithPrefix(tc.prefix, tc.envvars)
		if !reflect.DeepEqual(actual, tc.expected) {
			t.Fatalf("testcase #%d failed\n   actual value: '%s'\ndoes not match expected: '%s'\n", idx, actual, tc.expected)
		}

	}
}

func Test_addLabels(t *testing.T) {

	type testcase struct {
		labels   []string
		input    string
		expected string
	}
	testcases := []testcase{
		{
			labels: []string{`label1="L1a"`, `label2="L2a"`},
			input: `
# comment 1

foo_bar 20.1
foo_baz 20.2
bin_bar{existing="E1"} 20.3
`,
			expected: `
# comment 1

foo_bar{label1="L1a", label2="L2a"} 20.1
foo_baz{label1="L1a", label2="L2a"} 20.2
bin_bar{existing="E1", label1="L1a", label2="L2a"} 20.3
`,
		},
	}

	for idx, tc := range testcases {
		actualBS := addLabels([]byte(tc.input), tc.labels)
		actual := string(actualBS)
		if actual != tc.expected {
			t.Fatalf("testcase #%d failed\n   actual value: '%s'\ndoes not match expected: '%s'\n", idx, actual, tc.expected)
		}
	}
}
