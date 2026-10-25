package logx

import (
	"errors"
	"strings"
	"testing"
	"willshark/pkg/errs"
)

func TestFuncName(t *testing.T) {
	if GetSelfFuncName() != "TestFuncName" {
		t.Fatal(GetSelfFuncName())
	}
	if err := errs.WithMessage(errors.New("x"), "ctx"); !strings.Contains(err.Error(), "ctx: x") {
		t.Fatal(err)
	}
	ErrorLog(nil)
}

func TestErrorLogNonNil(t *testing.T) {
	ErrorLog(errors.New("logged"))
}
