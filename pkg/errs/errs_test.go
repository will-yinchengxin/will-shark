package errs

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestWrapKeepsIdentityAndCause(t *testing.T) {
	cause := errors.New("db down")
	err := fmt.Errorf("query: %w", Internal.Wrap(cause))

	if !errors.Is(err, Internal) {
		t.Fatal("wrapped error should match Internal by code")
	}
	if errors.Is(err, NotFound) {
		t.Fatal("must not match a different code")
	}
	if !errors.Is(err, cause) {
		t.Fatal("cause should be reachable via errors.Is")
	}
	if Internal.Unwrap() != nil {
		t.Fatal("Wrap must not mutate the shared predefined error")
	}
}

func TestWithMsgCopies(t *testing.T) {
	e := InvalidParam.WithMsg("手机号格式错误")
	if e.Msg != "手机号格式错误" || InvalidParam.Msg != "参数错误" {
		t.Fatal("WithMsg must return a copy")
	}
	if !errors.Is(e, InvalidParam) {
		t.Fatal("WithMsg copy should keep the code")
	}
}

func TestFrom(t *testing.T) {
	if From(nil) != nil {
		t.Fatal("From(nil) should be nil")
	}
	e := From(errors.New("boom"))
	if e.Code != Internal.Code || e.Status != http.StatusInternalServerError || e.Msg != Internal.Msg {
		t.Fatalf("plain error should map to Internal, got %+v", e)
	}
	if From(fmt.Errorf("x: %w", NotFound)).Code != NotFound.Code {
		t.Fatal("From should unwrap to the *Error")
	}
}
