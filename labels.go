package main

import (
	"fmt"
	"strings"

	log "github.com/Sirupsen/logrus"
)

func getEnvVarsWithPrefix(prefix string, kvs []string) []string {
	filtered := make([]string, 0)
	for _, kv := range kvs {
		if strings.HasPrefix(kv, prefix) {
			parts := strings.SplitN(kv, "=", 2)
			if len(parts) < 2 {
				continue
			}
			parts[0] = parts[0][len(prefix):]

			fullstring := fmt.Sprintf("%s=\"%s\"", parts[0], parts[1])
			log.Debug(fullstring)
			filtered = append(filtered, fullstring)
		}
	}
	return filtered
}

func addLabels(metrics []byte, labels []string) []byte {
	labelsStr := strings.Join(labels, ", ")

	lines := strings.Split(string(metrics), "\n")
	for i := 0; i < len(lines); i++ {
		lines[i] = lines[i]
		// skip comments and empty lines
		if (len(lines[i]) == 0) || (lines[i][0] == '#') {
			continue
		}

		lines[i] = strings.Replace(lines[i], "instance=", "original_instance=", -1)
		lines[i] = strings.Replace(lines[i], "job=", "original_job=", -1)

		// some metrics do not have labels and curly braces
		// find closing curly bracket - metrics that have labels
		cbPos := strings.LastIndex(lines[i], "}")
		if cbPos != -1 {
			//lines[i]= fmt.Sprintf("%s,%s", lines[i][:cbPos] )
			lines[i] = lines[i][:cbPos] + ", " + labelsStr + "" + lines[i][cbPos:]
		} else {
			spPos := strings.Index(lines[i], " ")
			if spPos != -1 {
				lines[i] = lines[i][:spPos] + "{" + labelsStr + "}" + lines[i][spPos:]
			}
		}
		lines[i] = strings.Replace(lines[i], ",,", ",", -1)
	}
	return []byte(strings.Join(lines, "\n"))
}
