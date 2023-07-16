package helpers

import (
	"fmt"

	"github.com/google/uuid"
)

func ConvertStringsToUUIDs(values []string) ([]uuid.UUID, error) {
	converted := make([]uuid.UUID, 0, len(values))

	for _, v := range values {
		parsed, err := uuid.Parse(v)
		if err != nil {
			return nil, fmt.Errorf("unable parse UUID '%s': %w", v, err)
		}

		converted = append(converted, parsed)
	}

	return converted, nil
}
