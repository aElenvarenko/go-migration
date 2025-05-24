package migration

import (
	"regexp"
	"strings"
)

// Find entry by expression return founded entry or empty string
func FindEntry(exp, content string) string {
	regExp := regexp.MustCompile(exp)

	allSubMatches := regExp.FindAllStringSubmatch(content, -1)
	if len(allSubMatches) > 0 {
		if len(allSubMatches[0]) >= 2 {
			return strings.Trim(allSubMatches[0][2], "\n")
		}
	}

	return ""
}
