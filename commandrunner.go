package commandrunner

import (
	"context"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// CommandSetupFunc defines a function type for perform additional setup
// on `cmdRef` before the runner invoke the command.
type CommandSetupFunc func(cmdRef *exec.Cmd) error

const (
	minimumCheckInterval = 100 * time.Millisecond
	minimumWaitInterval  = 200 * time.Millisecond
)

type runningInstance struct {
	instanceIndex   int
	cmdRef          *exec.Cmd
	timeoutAtNano   int64
	interruptAtNano int64
	terminateAtNano int64
}

type CommandRunner struct {
	lck      sync.Mutex
	runInsts []*runningInstance

	checkInterval    time.Duration
	stopWaitInterval time.Duration

	interruptWaitNano int64
	terminateWaitNano int64
	allStopAtNano     int64
}

func NewCommandRunner(
	maxRunningCommands int,
	checkInterval, stopWaitInterval, interruptWaitInterval, terminateWaitInterval time.Duration) (r *CommandRunner) {
	r = &CommandRunner{
		runInsts:          make([]*runningInstance, maxRunningCommands),
		checkInterval:     max(checkInterval, minimumCheckInterval),
		stopWaitInterval:  max(stopWaitInterval, minimumWaitInterval),
		interruptWaitNano: max(interruptWaitInterval, minimumWaitInterval).Nanoseconds(),
		terminateWaitNano: max(terminateWaitInterval, minimumWaitInterval).Nanoseconds(),
	}
	return
}

func (r *CommandRunner) stopAcceptingCommands() {
	r.lck.Lock()
	defer r.lck.Unlock()
	if r.allStopAtNano != 0 {
		return
	}
	r.allStopAtNano = time.Now().Add(r.stopWaitInterval).UnixNano()
}

func (r *CommandRunner) checkAllStopped() bool {
	r.lck.Lock()
	defer r.lck.Unlock()
	for _, inst := range r.runInsts {
		if inst != nil {
			return false
		}
	}
	return true
}

func (r *CommandRunner) checkIteration() {
	r.lck.Lock()
	defer r.lck.Unlock()
	currentTimeUnixNano := time.Now().UnixNano()
	normalRunning := (r.allStopAtNano == 0) || (currentTimeUnixNano < r.allStopAtNano)
	for _, inst := range r.runInsts {
		if (inst == nil) || (inst.timeoutAtNano == 0) {
			continue
		}
		if normalRunning && (currentTimeUnixNano < inst.timeoutAtNano) {
			continue
		}
		pgid := -(inst.cmdRef.Process.Pid)
		if inst.interruptAtNano == 0 {
			syscall.Kill(pgid, syscall.SIGINT)
			// TODO: record error
			inst.interruptAtNano = currentTimeUnixNano
		} else if (inst.terminateAtNano == 0) && (currentTimeUnixNano > (inst.interruptAtNano + r.interruptWaitNano)) {
			syscall.Kill(pgid, syscall.SIGTERM)
			// TODO: record error
			inst.terminateAtNano = currentTimeUnixNano
		} else if currentTimeUnixNano > (inst.terminateAtNano + r.terminateWaitNano) {
			syscall.Kill(pgid, syscall.SIGKILL)
			// TODO: record error
		}
	}
}

func (r *CommandRunner) checkLoop(ctx context.Context, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	ticker := time.NewTicker(r.checkInterval)
	defer ticker.Stop()
	acceptingRunCommands := true
	for acceptingRunCommands {
		select {
		case <-ctx.Done():
			r.stopAcceptingCommands()
			acceptingRunCommands = false
		case <-ticker.C:
			r.checkIteration()
		}
	}
	if r.checkAllStopped() {
		return
	}
	for range ticker.C {
		r.checkIteration()
		if r.checkAllStopped() {
			return
		}
	}
}

func (r *CommandRunner) StartRunner(ctx context.Context, waitGroup *sync.WaitGroup) {
	waitGroup.Add(1)
	go r.checkLoop(ctx, waitGroup)
}

func (r *CommandRunner) allocateRunningInstance() (instRef *runningInstance, err error) {
	r.lck.Lock()
	defer r.lck.Unlock()
	if r.allStopAtNano != 0 {
		err = ErrStopAcceptingCommands
		return
	}
	for idx, inst := range r.runInsts {
		if inst != nil {
			continue
		}
		instRef = &runningInstance{
			instanceIndex: idx,
		}
		r.runInsts[idx] = instRef
		return
	}
	err = ErrExceededMaxRunningCommands
	return
}

func (r *CommandRunner) releaseRunningInstance(instRef *runningInstance) {
	r.lck.Lock()
	defer r.lck.Unlock()
	r.runInsts[instRef.instanceIndex] = nil
}

func (r *CommandRunner) startRunningInstance(instRef *runningInstance, cmdRef *exec.Cmd, timeout time.Duration) (err error) {
	if err = cmdRef.Start(); nil != err {
		return
	}
	timeoutAtNano := time.Now().Add(timeout).UnixNano()
	r.lck.Lock()
	defer r.lck.Unlock()
	instRef.cmdRef = cmdRef
	instRef.timeoutAtNano = timeoutAtNano
	return
}

func (r *CommandRunner) Run(cmdName string, cmdArgs []string, cmdWorkDir string, cmdEnv []string, timeout time.Duration, setupFns ...CommandSetupFunc) (processState *os.ProcessState, err error) {
	instRef, err := r.allocateRunningInstance()
	if err != nil {
		return
	}
	defer r.releaseRunningInstance(instRef)
	cmdRef := exec.Command(cmdName, cmdArgs...)
	cmdRef.Dir = cmdWorkDir
	cmdRef.Env = cmdEnv
	for idx, setupFn := range setupFns {
		if err = setupFn(cmdRef); err != nil {
			err = newErrSetupCommandFailed(idx, err)
			return
		}
	}
	if cmdRef.SysProcAttr != nil {
		cmdRef.SysProcAttr.Setpgid = true
	} else {
		cmdRef.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true,
		}
	}
	if err = r.startRunningInstance(instRef, cmdRef, timeout); nil != err {
		return
	}
	err = cmdRef.Wait()
	processState = cmdRef.ProcessState
	return
}
