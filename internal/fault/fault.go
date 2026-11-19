// Package fault define categorias estáveis para consumidores da CLI.
package fault

import (
	"context"
	"errors"
	"fmt"
)

type Error struct {
	Code  string
	Exit  int
	Hint  string
	Cause error
}

func (e *Error) Error() string { return e.Cause.Error() }
func (e *Error) Unwrap() error { return e.Cause }
func New(code string, exit int, hint string, cause error) error {
	return &Error{code, exit, hint, cause}
}
func Usage(format string, args ...any) error {
	return New("invalid_argument", 2, "Consulte verman <comando> --help.", fmt.Errorf(format, args...))
}
func Classify(err error) *Error {
	if errors.Is(err, context.Canceled) {
		return &Error{"canceled", 130, "A operação foi cancelada; use recover se houver limpeza pendente.", err}
	}
	var typed *Error
	if errors.As(err, &typed) {
		return &Error{typed.Code, typed.Exit, typed.Hint, err}
	}
	return &Error{"operation_failed", 1, "Consulte verman doctor e a documentação de suporte.", err}
}
