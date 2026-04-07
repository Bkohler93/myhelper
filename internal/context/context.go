package context

import (
	"errors"
	"os"
)

// LoadContext reads context.md from the current working directory.
// Returns the file contents if the file exists, or an empty string if absent.
// Any error other than file-not-found is returned to the caller.
func LoadContext() (string, error) {
	data, err := os.ReadFile("context.md")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}
