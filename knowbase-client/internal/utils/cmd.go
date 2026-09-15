package utils

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
)

func RunCommand(cmd string, platform string) (string, error) {
	_cmdprg := "/bin/sh"
	_cmdarg := "-c"

	if platform == "windows" {
		_cmdprg = "cmd.exe"
		_cmdarg = "/C"
	}

	cmd = fmt.Sprintf("(%s) 2>&1", cmd)
	_cmd := exec.Command(_cmdprg, _cmdarg, cmd)
	_cmd.Env = os.Environ()

	var out bytes.Buffer
	var errOut bytes.Buffer
	_cmd.Stdout = &out
	_cmd.Stderr = &errOut

	err := _cmd.Start()
	if err != nil {
		return "", errors.New(errOut.String())
	}

	err = _cmd.Wait()
	if err != nil {
		return "", errors.New(errOut.String())
	}
	return out.String(), nil
}

func RunCommandWithChan(cmd string, platform string, ch chan string, exitCh chan error) *exec.Cmd {
	// init cmd object and redirect stederr to stdout
	_cmdprg := "/bin/sh"
	_cmdarg := "-c"

	if platform == "windows" {
		_cmdprg = "cmd.exe"
		_cmdarg = "/c"
	}

	cmd = fmt.Sprintf("(%s) 2>&1", cmd)
	_cmd := exec.Command(_cmdprg, _cmdarg, cmd)
	_cmd.Env = os.Environ()
	go func() {
		stdout, err := _cmd.StdoutPipe()
		if err != nil {
			exitCh <- err
			return
		}

		stderr, err := _cmd.StderrPipe()
		if err != nil {
			exitCh <- err
			return
		}

		// to start cmd
		if err = _cmd.Start(); err != nil {
			exitCh <- err
			return
		}

		// read stdout
		rd := bufio.NewReaderSize(stdout, 4*1024*1024)
		for {
			str, err := rd.ReadString('\n')
			if len(str) > 0 {
				ch <- str
			}
			if err != nil && err != bufio.ErrBufferFull {
				break
			}
		}

		// read stderr
		rd = bufio.NewReaderSize(stderr, 4*1024*1024)
		for {
			str, err := rd.ReadString('\n')
			if len(str) > 0 {
				ch <- str
			}
			if err != nil && err != bufio.ErrBufferFull {
				break
			}
		}

		// wait cmd to exit
		err = _cmd.Wait()
		if err != nil {
			//error_msg := cmd + ": " + err.Error()
			exitCh <- errors.New(err.Error())
		} else {
			exitCh <- nil
		}
	}()
	return _cmd
}
