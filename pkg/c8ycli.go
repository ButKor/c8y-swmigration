package main

import (
	"io"
	"log"
	"os/exec"
)

type Executor struct {
	Command string
	cmd     *exec.Cmd
	Options *CLIOptions
}

type CLIOptions struct {
	Host         string
	Username     string
	Password     string
	EnableCreate bool
	EnableUpdate bool
	EnableDelete bool
}

type ExecutorResult struct {
	exec.Cmd

	ExitCode int
	Stdout   []byte
	Stderr   []byte
}

func (e *Executor) prepare() *exec.Cmd {
	//cmd := exec.Command("/bin/bash", "-c", e.Command)
	cmd := exec.Command(e.Command)

	settings := []string{
		"C8Y_HOST=" + e.Options.Host,
		"C8Y_USER=" + e.Options.Username,
		"C8Y_PASSWORD=" + e.Options.Password,
		"C8Y_SETTINGS_MODE_CONFIRMATION=NONE",
		"C8Y_SETTINGS_ACTIVITYLOG_PATH=/tmp",
		"C8Y_SETTINGS_ACTIVITYLOG_ENABLED=false",
	}
	if e.Options.EnableCreate {
		settings = append(settings, "C8Y_SETTINGS_MODE_ENABLECREATE=true")
	}
	if e.Options.EnableUpdate {
		settings = append(settings, "C8Y_SETTINGS_MODE_ENABLEUPDATE=true")
	}
	if e.Options.EnableDelete {
		settings = append(settings, "C8Y_SETTINGS_MODE_ENABLEDELETE=true")
	}

	cmd.Env = settings
	e.cmd = cmd
	return e.cmd
}

func (e *Executor) execute() (*ExecutorResult, error) {
	e.prepare()

	cmd := e.cmd
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	err := cmd.Start()

	if err != nil {
		log.Fatal(err)
	}

	slurpStdout, _ := io.ReadAll(stdout)
	slurpStderr, _ := io.ReadAll(stderr)

	_ = cmd.Wait()

	return &ExecutorResult{
		Cmd:      *cmd,
		ExitCode: cmd.ProcessState.ExitCode(),
		Stdout:   slurpStdout,
		Stderr:   slurpStderr,
	}, nil
}

func (e *Executor) Execute(wait bool) (*ExecutorResult, error) {
	if wait {
		return e.execute()
	}
	go func() {
		_, _ = e.execute()
	}()
	return nil, nil
}
