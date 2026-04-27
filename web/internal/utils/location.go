package utils

import (
	"sort"
	"strings"
)

var provinceAliases = map[string]string{
	"on": "ontario",
	"ont": "ontario",
	"bc": "british columbia",
	"ab": "alberta",
	"sk": "saskatchewan",
	"mb": "manitoba",
	"qc": "quebec",
	"nb": "new brunswick",
	"ns": "nova scotia",
	"pe": "prince edward island",
	"nl": "newfoundland and labrador",
	"yt": "yukon",
	"nt": "northwest territories",
	"nu": "nunavut",
}

var gtaCities = []string{"mississauga", "toronto", "brampton", "vaughan", "markham", "richmond hill", "oakville", "burlington", "ajax", "pickering", "oshawa", "whitby"}

// BuildLocationSearch normalizes a location string into searchable tokens.
func BuildLocationSearch(location string) string {
	lower := strings.ToLower(strings.TrimSpace(location))
	tokens := make(map[string]struct{})

	for _, part := range strings.FieldsFunc(lower, func(r rune) bool {
		return r == ' ' || r == ',' || r == '-' || r == '/'
	}) {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		tokens[part] = struct{}{}
		if alias, ok := provinceAliases[part]; ok {
			tokens[alias] = struct{}{}
		}
	}

	// Add GTA synonyms
	for city := range tokens {
		for _, gta := range gtaCities {
			if city == gta {
				tokens["gta"] = struct{}{}
				tokens["greater"] = struct{}{}
				tokens["toronto"] = struct{}{}
				tokens["area"] = struct{}{}
				break
			}
		}
	}

	var result []string
	for t := range tokens {
		result = append(result, t)
	}
	sort.Strings(result)
	return strings.Join(result, " ")
}
