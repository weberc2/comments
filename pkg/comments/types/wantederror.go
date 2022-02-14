package types

import "fmt"

type WantedError interface {
	CompareErr(error) error
}

type NilError struct{}

func (NilError) CompareErr(other error) error {
	if other == nil {
		return nil
	}
	return fmt.Errorf("wanted `nil`; found `%T`: %v", other, other)
}
