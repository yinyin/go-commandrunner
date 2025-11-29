package commandrunner

import (
	"errors"
	"strconv"
)

var ErrExceededMaxRunningCommands = errors.New("exceeded maximum number of running commands")

type ErrSetupCommandFailed struct {
	SetupFuncIndex int
	Err            error
}

func newErrSetupCommandFailed(setupFuncIndex int, err error) *ErrSetupCommandFailed {
	return &ErrSetupCommandFailed{
		SetupFuncIndex: setupFuncIndex,
		Err:            err,
	}
}

func (e *ErrSetupCommandFailed) Error() string {
	return "failed at [" + strconv.FormatInt(int64(e.SetupFuncIndex), 10) + "-th] command setup function: " + e.Err.Error()
}

func (e *ErrSetupCommandFailed) Unwrap() error {
	return e.Err
}
