package commandrunner

import (
	"encoding/json"
	"os"
	"os/exec"
)

type commandInfo struct {
	ExePath string   `json:"exe_path"`
	Args    []string `json:"args"`
	WorkDir string   `json:"workdir"`
	Env     []string `json:"env"`
}

func newCommandInfo(cmdRef *exec.Cmd) (info *commandInfo) {
	info = &commandInfo{
		ExePath: cmdRef.Path,
		Args:    cmdRef.Args,
		WorkDir: cmdRef.Dir,
		Env:     cmdRef.Env,
	}
	return
}

func logCommandInfo(fp *os.File, cmdRef *exec.Cmd) (err error) {
	cmdInfo := newCommandInfo(cmdRef)
	enc := json.NewEncoder(fp)
	enc.SetIndent("", "  ")
	err = enc.Encode(cmdInfo)
	return
}
