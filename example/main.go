package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
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
	waitGroup.Add(2)
	go cmd1(&waitGroup, cmdrunner, cmdEnv)
	go cmd2(&waitGroup, cmdrunner, cmdEnv)
	slog.InfoContext(ctx, "wait runner")
	waitGroup.Wait()
}
