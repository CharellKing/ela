package utils

import (
	"errors"
	"fmt"
	"strings"
)

// Errs is an error that collects other errors, for when you want to do
// several things and then report all of them.
type Errs struct {
	errors []error
}

func (e *Errs) Add(err error) {
	if err != nil {
		e.errors = append(e.errors, err)
	}
}

func (e *Errs) Ret() error {
	if e == nil || e.IsEmpty() {
		return nil
	}
	return e
}

func (e *Errs) IsEmpty() bool {
	return e.Len() == 0
}

func (e *Errs) Len() int {
	return len(e.errors)
}

func (e *Errs) Error() string {
	asStr := make([]string, len(e.errors))
	for i, x := range e.errors {
		asStr[i] = x.Error() + "\n" + fmt.Sprintf("%+v", x) // 打印错误和栈信息
	}
	return strings.Join(asStr, ". ")
}

func (e *Errs) Is(target error) bool {
	for _, candidate := range e.errors {
		if errors.Is(candidate, target) {
			return true
		}
	}
	return false
}

func (e *Errs) As(target interface{}) bool {
	for _, candidate := range e.errors {
		if errors.As(candidate, &target) {
			return true
		}
	}
	return false
}
