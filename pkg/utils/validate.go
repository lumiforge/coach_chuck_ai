package utils

import "fmt"

func ValidateAllowedStrings(values []string, allowed map[string]struct{}, fieldName string) error {
	for _, value := range values {
		if _, ok := allowed[value]; !ok {
			return fmt.Errorf("invalid %s value: %q", fieldName, value)
		}
	}
	return nil
}
