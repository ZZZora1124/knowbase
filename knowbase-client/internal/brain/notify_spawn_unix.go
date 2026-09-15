// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

//go:build !windows

package brain

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"icode.baidu.com/baidu/devops-ai/knowbase-client/pkg/types"
)

// buildDuccCmd 构造派生 ducc 子进程的 *exec.Cmd。提取为独立函数便于单测。
// sync=true 时把 stdout/stderr 接到父进程，使调用方（如 phase4_dispatch_sidecar.py
// 用 subprocess.run capture_output 包住 `knowbase brain notify --sync`）能拿到
// signal-detector 的 "信号：..." 日志行；async 模式保持原行为（detach 后悄悄跑）。
func buildDuccCmd(duccBin, systemPrompt, tmpFile, notebookUUID, authToken,
	allowedTools, model string, sync bool, chatMode string) *exec.Cmd {

	promptFinal := systemPrompt
	if chatMode == types.ChatModeGroup {
		promptFinal = systemPrompt + "\n\n" + groupModePromptFragment
	}

	args := []string{"--print", tmpFile}
	for _, d := range computeAddDirs() {
		args = append(args, "--add-dir", d)
	}
	if model != "" {
		args = append(args, "--model", model)
	}
	args = append(args,
		"--append-system-prompt", promptFinal,
		"--app-source", "dodo-brain",
		"--allowedTools", allowedTools,
	)
	cmd := exec.Command(duccBin, args...)
	env := append(os.Environ(),
		"KNOWBASE_BRAIN_UUID="+notebookUUID,
		"COMATE_AUTH_TOKEN="+authToken,
	)
	if os.Getenv(types.EnvKnowbaseEnv) == "test" {
		env = append(env, "BAIDU_CC_SERVER_ENV=test")
	}
	cmd.Env = env
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if sync {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	return cmd
}

// spawnDucc 在 Unix 上以新会话 detach 派生 ducc 子进程。
// sync=true 时阻塞等待子进程结束后再返回（用于 --sync 串行派发场景）。
func spawnDucc(duccBin, systemPrompt, tmpFile, notebookUUID, authToken,
	allowedTools, model string, sync bool, chatMode string) error {
	cmd := buildDuccCmd(duccBin, systemPrompt, tmpFile, notebookUUID, authToken, allowedTools, model, sync, chatMode)
	// 在 ~/ark-brain 目录下执行，避免 stop hook 循环触发
	if err := ensureWorkDir(cmd); err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start ducc subprocess: %w", err)
	}
	if sync {
		if err := cmd.Wait(); err != nil {
			return fmt.Errorf("ducc subprocess exited with error: %w", err)
		}
	}
	return nil
}
