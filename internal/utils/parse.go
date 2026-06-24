package utils

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

func ParseFloat32Ptr(raw string) (*float32, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	v, err := strconv.ParseFloat(raw, 32)
	if err != nil {
		return nil, err
	}

	if math.IsNaN(v) || math.IsInf(v, 0) {
		return nil, fmt.Errorf("value must be finite")
	}

	f := float32(v)

	return &f, nil
}

func ParseNonnegativeInt32Ptr(raw string) (*int32, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	v, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return nil, err
	}

	if v < 0 {
		return nil, fmt.Errorf("value must be nonnegative")
	}

	i := int32(v)

	return &i, nil
}
