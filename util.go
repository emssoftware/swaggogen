package main

import "strings"

func sContains(set []string, s string) bool {
	for _, s_ := range set {
		if s_ == s {
			return true
		}
	}

	return false
}

func shouldIgnore(path string) bool {
	for _, ignored := range ignoredPackages {
		if strings.Contains(path, ignored) {
			return true
		}
	}
	return false
}
