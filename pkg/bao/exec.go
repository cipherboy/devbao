package bao

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

const (
	SERVICE_LOG_NAME = "service.log"
	EXEC_JSON_NAME   = "exec.json"
)

type ExecEnvironment struct {
	Binary    string   `json:"binary"`
	Args      []string `json:"args"`
	Directory string   `json:"directory"`

	ConnectAddress string `json:"connection_address"`

	Pid int `json:"pid"`
}

func (e *ExecEnvironment) Kill() error {
	if e.Pid == 0 {
		return nil
	}

	if err := e.ValidateRunning(); err != nil {
		// Any error in validating the process means that it has already
		// exited so hide this error.
		e.Pid = 0
		return nil
	}

	proc, err := process.NewProcess(int32(e.Pid))
	if err != nil {
		return fmt.Errorf("failed find process with pid (%d): %w", e.Pid, err)
	}

	killErr := proc.Kill()
	termErr := proc.Terminate()

	if killErr != nil && termErr != nil {
		return fmt.Errorf("failed to kill process (%d):\n\twhile killing: %w\n\twhile terminating: %v", e.Pid, killErr, termErr)
	}

	return nil
}

func (e *ExecEnvironment) ValidateRunning() error {
	proc, err := process.NewProcess(int32(e.Pid))
	if err != nil {
		return fmt.Errorf("failed find process with pid (%d): %w", e.Pid, err)
	}

	binary, err := proc.Exe()
	if err != nil {
		return fmt.Errorf("failed to get cli path of process %d (is it running?): %w", e.Pid, err)
	}

	eBinary := filepath.Base(e.Binary)
	binary = filepath.Base(binary)

	if (eBinary != "" && binary != eBinary) || (eBinary == "" && binary != "vault" && binary != "bao") {
		return fmt.Errorf("process with pid %d is no longer the original process: different binary paths", e.Pid)
	}

	return nil
}

func (e *ExecEnvironment) SaveConfig(pid int) error {
	e.Pid = pid

	if err := e.ValidateRunning(); err != nil {
		return fmt.Errorf("failed to ensure instance was started correctly: %w\n\tUsually this means that the server bound to this port was not started by us.", err)
	}

	path := filepath.Join(e.Directory, EXEC_JSON_NAME)
	configFile, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open config file (`%v`) for writing: %w", path, err)
	}

	defer configFile.Close()

	if err := json.NewEncoder(configFile).Encode(e); err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return nil
}

func (e *ExecEnvironment) WaitAlive(logPath string) error {
	// Wait before checking alive status: if a listener conflicts due to
	// already bound port, we want to give the command time to exit.
	time.Sleep(100 * time.Millisecond)

	ok := false
	for i := 0; i < 50; i++ {
		con, err := net.Dial("tcp", e.ConnectAddress)
		if err != nil {
			time.Sleep(200 * time.Millisecond)
			continue
		}

		_ = con.Close()
		ok = true
	}

	if !ok {
		return fmt.Errorf("failed to connect to server's listener (%v); check error logs at %v", e.ConnectAddress, logPath)
	}

	return nil
}

func (e *ExecEnvironment) ReadLogs() (string, error) {
	logPath := filepath.Join(e.Directory, SERVICE_LOG_NAME)
	logFile, err := os.Open(logPath)
	if err != nil {
		return "", fmt.Errorf("failed to open logs (%v) for reading: %w", logPath, err)
	}

	rawLogs, err := io.ReadAll(logFile)
	if err != nil {
		return "", fmt.Errorf("failed to read logs (%v): %w", logPath, err)
	}

	// Since the server may still be actively writing to the logs,
	// if the last character is not a new line, append one of our own
	// and indicate the logs were truncated.
	logs := string(rawLogs)
	if len(logs) > 0 && logs[len(logs)-1] != '\n' {
		logs += "\n\t(logs truncated)\n"
	} else if len(logs) == 0 {
		logs += "\n\t(no logs)\n"
	}

	return logs, nil
}

func Exec(env *ExecEnvironment) error {
	binary, err := findBestBinary()
	if err != nil {
		return err
	}

	env.Binary = binary
	return doExec(env)
}

func ExecBao(env *ExecEnvironment) error {
	binary, err := expandBinary("openbao")
	if err != nil {
		binary, err = expandBinary("bao")
	}
	if err != nil {
		return err
	}

	env.Binary = binary
	return doExec(env)
}

func ExecVault(env *ExecEnvironment) error {
	binary, err := expandBinary("vault")
	if err != nil {
		return err
	}

	env.Binary = binary
	return doExec(env)
}

func findBestBinary() (string, error) {
	binary, err := expandBinary("openbao")
	if err == nil {
		return binary, nil
	}

	binary, err = expandBinary("bao")
	if err == nil {
		return binary, nil
	}

	return expandBinary("vault")
}

func expandBinary(ref string) (string, error) {
	env := fmt.Sprintf("%s_BINARY", strings.ToUpper(ref))
	val := os.Getenv(env)
	if val != "" {
		ref = val
	}

	path, err := exec.LookPath(ref)
	if err != nil {
		if errors.Is(err, exec.ErrDot) {
			return filepath.Abs(path)
		}

		return "", err
	}

	return path, nil
}

func expandErrWithLogs(env *ExecEnvironment, err error) error {
	logs, logsErr := env.ReadLogs()

	if logsErr != nil {
		err = fmt.Errorf("%w\n\t(failed to read logs: %v)", err, logsErr)
	} else {
		err = fmt.Errorf("%w\n\n==== SERVER LOGS ====\n\n%v", err, logs)
	}

	return err
}

func doExec(env *ExecEnvironment) error {
	logPath := filepath.Join(env.Directory, SERVICE_LOG_NAME)
	logs, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open logs (`%v`) for writing: %w", logPath, err)
	}
	defer logs.Close()

	cmd := exec.Command(env.Binary, env.Args...)
	cmd.Dir = env.Directory
	cmd.Stdout = logs
	cmd.Stderr = cmd.Stdout

	cli := append([]string{env.Binary}, env.Args...)

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start server (cli: %v): %w", cli, err)
	}

	if err := env.WaitAlive(logPath); err != nil {
		err = fmt.Errorf("failed to wait for listener (%v) to come up (cli: %v): %w", env.ConnectAddress, cli, err)
		return expandErrWithLogs(env, err)
	}

	if err := env.SaveConfig(cmd.Process.Pid); err != nil {
		err = fmt.Errorf("failed to save exec config (cli: %v): %w", cli, err)
		return expandErrWithLogs(env, err)
	}

	cmd.Process.Release()

	return nil
}
