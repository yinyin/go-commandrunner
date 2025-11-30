package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"

	commandrunner "github.com/yinyin/go-commandrunner"
)

const defaultEnvPath = "/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"

var inheritEnvKeys = []string{
	"HOME",
	"USER",
	"LOGNAME",
}

func cmd1(waitGroup *sync.WaitGroup, cmdrunner *commandrunner.CommandRunner, cmdEnv []string) {
	defer waitGroup.Done()
	slog.Info("run c")
	termLog, err := commandrunner.OpenCombinedOutputFile("/tmp/go-cmdrun-1", 0600)
	if err != nil {
		slog.Error("cannot open terminal log for cmd1", "error", err)
		return
	}
	defer termLog.Close()
	processState, err := cmdrunner.Run("sleep", []string{"5"}, "/tmp", cmdEnv, time.Second*7, termLog.SetupCommand)
	termLog.LogResult(processState, err)
	if err != nil {
		slog.Error("run cmd1 failed", "error", err)
		return
	}
	slog.Info("run cmd1 completed", "processState", processState)
}

func cmd2(waitGroup *sync.WaitGroup, cmdrunner *commandrunner.CommandRunner, cmdEnv []string) {
	defer waitGroup.Done()
	termLog, err := commandrunner.OpenCombinedOutputFile("/tmp/go-cmdrun-2", 0600)
	if err != nil {
		slog.Error("cannot open terminal log for cmd2", "error", err)
		return
	}
	defer termLog.Close()
	processState, err := cmdrunner.Run("sleep", []string{"10"}, "/tmp", cmdEnv, time.Second*6, termLog.SetupCommand)
	termLog.LogResult(processState, err)
	if err != nil {
		slog.Error("run cmd2 failed", "error", err)
		return
	}
	slog.Info("run cmd2 completed", "processState", processState)
}

func cmd3(waitGroup *sync.WaitGroup, cmdrunner *commandrunner.CommandRunner, cmdEnv []string) {
	defer waitGroup.Done()
	termLog, err := commandrunner.OpenCombinedOutputFile("/tmp/go-cmdrun-3", 0600)
	if err != nil {
		slog.Error("cannot open terminal log for cmd3", "error", err)
		return
	}
	defer termLog.Close()
	processState, err := cmdrunner.Run("sleep", []string{"20"}, "/tmp", cmdEnv, time.Second*6, termLog.SetupCommand)
	termLog.LogResult(processState, err)
	if err != nil {
		slog.Error("run cmd3 failed", "error", err)
		return
	}
	slog.Info("run cmd3 completed", "processState", processState)
}

func cmdSleep(waitGroup *sync.WaitGroup, cmdrunner *commandrunner.CommandRunner, cmdEnv []string, runSeconds, timeoutSeconds int64) {
	defer waitGroup.Done()
	runSecondsText := strconv.FormatInt(runSeconds, 10)
	timeoutSecondsText := strconv.FormatInt(timeoutSeconds, 10)
	slog.Info("run command cmd-sleep-" + runSecondsText + "-tm" + timeoutSecondsText)
	termLog, err := commandrunner.OpenCombinedOutputFile("/tmp/go-cmdrun-sleep-"+runSecondsText+"-tm"+timeoutSecondsText, 0600)
	if err != nil {
		slog.Error("cannot open terminal log for cmd-sleep-"+runSecondsText+"-tm"+timeoutSecondsText, "error", err)
		return
	}
	defer termLog.Close()
	processState, err := cmdrunner.Run("sleep", []string{runSecondsText}, "/tmp", cmdEnv, time.Second*time.Duration(timeoutSeconds), termLog.SetupCommand)
	termLog.LogResult(processState, err)
	if err != nil {
		slog.Error("run cmd-sleep-"+runSecondsText+"-tm"+timeoutSecondsText+" failed", "error", err)
		return
	}
	slog.Info("run cmd-sleep-"+runSecondsText+"-tm"+timeoutSecondsText+" completed", "processState", processState)
}

func main() {
	cmdEnv := make([]string, 0, len(inheritEnvKeys)+1)
	for _, envKey := range inheritEnvKeys {
		if envValue := os.Getenv(envKey); envValue != "" {
			cmdEnv = append(cmdEnv, envKey+"="+envValue)
		}
	}
	cmdEnv = append(cmdEnv, "PATH="+defaultEnvPath)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	var waitGroup sync.WaitGroup
	cmdrunner := commandrunner.NewCommandRunner(2, time.Second*3, time.Second*5, time.Second*2, time.Second*5)
	cmdrunner.StartRunner(ctx, &waitGroup)
	waitGroup.Add(4)
	go cmdSleep(&waitGroup, cmdrunner, cmdEnv, 5, 7)
	go cmdSleep(&waitGroup, cmdrunner, cmdEnv, 10, 6)
	time.Sleep(time.Second * 8) // wait for cmd1 stops
	go cmdSleep(&waitGroup, cmdrunner, cmdEnv, 30, 20)
	go cmdSleep(&waitGroup, cmdrunner, cmdEnv, 2, 6)
	slog.InfoContext(ctx, "wait runner (press Ctrl+C to test graceful stop)...")
	waitGroup.Wait()
}
