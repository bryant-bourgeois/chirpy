package main

import (
	"strings"
)

func cleanProfanity(str string) string {
	badWords := []string{"kerfuffle", "sharbert", "fornax"}
	lower := strings.ToLower(str)
	for _, val := range badWords {
		idx := strings.Index(lower, val)
		if idx == -1 {
			continue
		} else {
			str = str[:idx] + "****" + str[idx+len(val):]
			lower = lower[:idx] + "****" + lower[idx+len(val):]
		}
	}
	return str
}
