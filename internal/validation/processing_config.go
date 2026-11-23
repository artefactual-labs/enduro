package validation

import (
	"fmt"
	"unicode/utf8"
)

// MaxProcessingConfigNameLength matches the collection.processing_config column.
const MaxProcessingConfigNameLength = 255

func ValidateProcessingConfigName(name string) error {
	if utf8.RuneCountInString(name) > MaxProcessingConfigNameLength {
		return fmt.Errorf("processing configuration name must be at most %d characters", MaxProcessingConfigNameLength)
	}
	return nil
}
