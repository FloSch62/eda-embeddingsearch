package utils

// Contains checks if a slice contains a string
func Contains(tokens []string, word string) bool {
	for _, t := range tokens {
		if t == word {
			return true
		}
	}
	return false
}
