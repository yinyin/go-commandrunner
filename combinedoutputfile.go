package commandrunner

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

type CombinedOutputFile struct {
	fp      *os.File
	startAt time.Time
}

func OpenCombinedOutputFile(termLogPath string, perm os.FileMode) (f *CombinedOutputFile, err error) {
	fp, err := os.OpenFile(termLogPath, os.O_WRONLY|os.O_CREATE, perm)
	if err != nil {
		return
	}
	f = &CombinedOutputFile{
		fp: fp,
	}
	return
}

func (f *CombinedOutputFile) SetupCommand(cmdRef *exec.Cmd) (err error) {
	if f.fp == nil {
		return
	}
	if err = logCommandInfo(f.fp, cmdRef); nil != err {
		err = fmt.Errorf("failed to log command info: %w", err)
		return
	}
	f.startAt = time.Now()
	fmt.Fprintf(f.fp, "\n-- (start: %v) --------\n\n", f.startAt)
	cmdRef.Stdout = f.fp
	cmdRef.Stderr = f.fp
	return
}

func (f *CombinedOutputFile) LogResult(processState *os.ProcessState, runErr error) {
	if f.fp == nil {
		return
	}
	completeAt := time.Now()
	runCost := completeAt.Sub(f.startAt)
	fmt.Fprintf(f.fp, "\n\n-- (complete: %v, cost: %v) --------\n", completeAt, runCost)
	fmt.Fprintf(f.fp, "Result: %v (error=%v)\n", processState, runErr)
}

func (f *CombinedOutputFile) Close() (err error) {
	if f.fp == nil {
		return
	}
	err = f.fp.Close()
	f.fp = nil
	return
}
