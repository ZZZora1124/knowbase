// Copyright 2025 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package update

import (
	"context"
	"errors"
	"fmt"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/utils"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/pkg/types"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gofrs/flock"
)

// ============================================================================
// Auto Update Functions
// ============================================================================

const (
	MaxLockRetries    = 5
	LockRetryInterval = 3 * time.Second
	DownloadTimeout   = 5 * time.Minute
)

type Update struct {
	OS               string
	Arch             string
	BosRoot          string
	BosPackageName   string
	BosVersionFile   string
	LocalPackageDir  string
	LocalVersionFile string
	HomeKnowbasePath string
	LockFilePath     string
}

// Init function
// Args:
//
//	home_dir: path of home
func (update *Update) Init(baseDir string) {
	// current platform os and arch
	update.OS = runtime.GOOS
	update.Arch = runtime.GOARCH
	update.LocalPackageDir = baseDir
	update.LocalVersionFile = filepath.Join(update.LocalPackageDir, "knowbase", "version")
	update.HomeKnowbasePath = filepath.Join(update.LocalPackageDir, "knowbase", "bin", "knowbase")
	update.LockFilePath = filepath.Join(update.LocalPackageDir, ".lock")
	update.BosRoot = update.GetBosRoot()
	update.BosPackageName = "knowbase-%s-%s-%s.tar.bz2"
	update.BosVersionFile = fmt.Sprintf("%s/%s", update.BosRoot, "knowbase_latest_version.txt")
}

// checkAndPerformUpdate checks for updates and performs update if needed
func (update *Update) CheckAndPerformUpdate() error {
	// version check
	current_version, err := update.GetCurrentVersion()
	if err != nil {
		// 获取当前版本失败不需要退出，继续执行更新
		utils.PrintlnError("Fail to get current client version: %s", err.Error())
	}

	latest_version, err := update.GetLatestVersion()
	if err != nil {
		err_msg := fmt.Sprintf("Fail to get latest client version: %s", err.Error())
		return errors.New(err_msg)
	}

	if current_version == latest_version {
		return nil
	}

	// Acquire file lock for update
	lock, err := update.AcquireFileLock()
	if err != nil {
		return fmt.Errorf("failed to acquire file lock: %v", err)
	}
	defer lock.Unlock()

	// Handle SIGTSTP signal (cross-platform)
	ignoreSigtstp()
	resetSigtstp()

	// Check whether client has been updated by another process
	current_version, err = update.GetCurrentVersion()
	if err != nil {
		err_msg := fmt.Sprintf("Fail to get current client version: %s", err.Error())
		return errors.New(err_msg)
	}

	if current_version == latest_version {
		return nil
	}

	fmt.Printf("Try to update knowbase client, current version is [%s] ...\n", current_version)
	err = update.UpdateClient(current_version, latest_version)
	if err != nil {
		err_msg := fmt.Sprintf("Fail to update client: %s", err.Error())
		return errors.New(err_msg)
	}
	fmt.Printf("knowbase client updates from [%s] to [%s] successfully!\n", current_version, latest_version)

	// 更新完成后手动释放一下文件锁
	if err := lock.Unlock(); err != nil {
		utils.PrintlnError("Failed to release file lock: %v", err)
	}

	update.RestartClient()
	return nil
}

// GetCurrentVersion gets the local current version information
func (update *Update) GetCurrentVersion() (string, error) {
	// read version file
	versionPath := update.LocalVersionFile
	data, err := os.ReadFile(versionPath)
	if err != nil {
		return "", err
	}

	// parse version
	version := strings.TrimSpace(string(data))
	return version, nil
}

// restart service
func (update *Update) RestartClient() {
	// 直接使用 exec.Command 并传递参数数组，避免 shell 解析特殊字符
	cmd := exec.Command(update.HomeKnowbasePath, os.Args[1:]...)

	// 继承标准输入、输出和错误
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// 继承所有环境变量
	cmd.Env = os.Environ()

	err := cmd.Run()
	if err != nil {
		os.Exit(types.ExitRunSubCMDError)
	}
	os.Exit(types.ExitSuccess)
}

// getUpdateInfo gets update information from DepServer
func (update *Update) GetLatestVersion() (string, error) {
	// 通过http请求获取，以实现跨平台
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", update.BosVersionFile, nil)
	if err != nil {
		return "", err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(body)), nil
}

// AcquireFileLock acquires exclusive file lock and returns the lock object
func (update *Update) AcquireFileLock() (*flock.Flock, error) {
	lock := flock.New(update.LockFilePath)

	for i := 0; i < MaxLockRetries; i++ {
		locked, err := lock.TryLock()
		if err != nil {
			return nil, fmt.Errorf("lock operation failed: %v", err)
		}

		if locked {
			return lock, nil
		}

		if i < MaxLockRetries-1 {
			fmt.Printf("Cannot acquire lock, retrying in %v...\n", LockRetryInterval)
			time.Sleep(LockRetryInterval)
		}
	}

	return nil, fmt.Errorf("failed to acquire lock after %d retries", MaxLockRetries)
}

// update client
// Args:
//
//	current_version: current client version
//	latest_version : latest client version
//
// Returns:
//
//	nil stands for success, otherwise stands for fail
func (update *Update) UpdateClient(current_version string, latest_version string) (err error) {
	// 1. download new client package to local dir
	if !utils.FileExists(update.LocalPackageDir) {
		err = os.MkdirAll(update.LocalPackageDir, 0777)
		if err != nil {
			err_msg := fmt.Sprintf("Fail to create local package dir [%s]: %s", update.LocalPackageDir, err.Error())
			return errors.New(err_msg)
		}
	}
	bos_package_name := fmt.Sprintf(update.BosPackageName, update.OS, update.Arch, latest_version)
	local_package_path := filepath.Join(update.LocalPackageDir, bos_package_name)
	if utils.FileExists(local_package_path) {
		err = os.RemoveAll(local_package_path)
		if err != nil {
			err_msg := fmt.Sprintf("Fail to remove old package [%s]: %s", local_package_path, err.Error())
			return errors.New(err_msg)
		}
	}
	bos_package_url := fmt.Sprintf("%s/%s", update.BosRoot, bos_package_name)
	// wget -O ~/.knowbase/knowbase_linux_amd64_1.0.0.tar.bz2 https://knowbase-client.bj.bcebos.com/knowbase-client/knowbase_linux_amd64_1.0.0.tar.bz2
	cmd := fmt.Sprintf("wget -O %s %s --no-check-certificate", local_package_path, bos_package_url)

	// 下载超时时间
	timeout_ticker := time.NewTicker(DownloadTimeout)
	defer timeout_ticker.Stop()

	// 下载过程中，每隔 10s 打印一次日志提示
	interval_time_s := 10
	interval_ticker := time.NewTicker(time.Duration(interval_time_s) * time.Second)
	defer interval_ticker.Stop()

	msg_ch := make(chan string)
	exit_ch := make(chan error)
	utils.RunCommandWithChan(cmd, update.OS, msg_ch, exit_ch)

	download_start_time := time.Now()

	running := true
	for running {
		select {
		// 注意: case <- msg_ch 不能删，否则会被阻塞
		case <-msg_ch:
		case err = <-exit_ch:
			running = false
		default:
		}

		select {
		case <-timeout_ticker.C:
			err_msg := fmt.Sprintf("Fail to download new client package, timeout(%v seconds)", DownloadTimeout)
			err = errors.New(err_msg)
			running = false
		case <-interval_ticker.C:
			download_take_s := time.Since(download_start_time).Seconds()
			info_msg := fmt.Sprintf("Download new client package takes: %.3fs\n", download_take_s)
			fmt.Println(info_msg)
		default:
		}
	}

	if err != nil {
		err_msg := fmt.Sprintf("Fail to download new client package: %s", err.Error())
		return errors.New(err_msg)
	}

	// 2. untar new client package to tmp
	tmp_dir := filepath.Join(update.LocalPackageDir, "tmp")
	if !utils.FileExists(tmp_dir) {
		err := os.MkdirAll(tmp_dir, 0777)
		if err != nil {
			err_msg := fmt.Sprintf("Fail to create tmp dir [%s]: %s", tmp_dir, err.Error())
			return errors.New(err_msg)
		}
	}

	// Create a subdirectory in tmp for extraction
	new_knowbase := fmt.Sprintf("knowbase-%s-%s-%s", update.OS, update.Arch, latest_version)
	tmp_extract_dir := filepath.Join(tmp_dir, new_knowbase)
	if utils.FileExists(tmp_extract_dir) {
		err = os.RemoveAll(tmp_extract_dir)
		if err != nil {
			err_msg := fmt.Sprintf("Fail to remove old extract dir [%s]: %s", tmp_extract_dir, err.Error())
			return errors.New(err_msg)
		}
	}
	err = os.MkdirAll(tmp_extract_dir, 0777)
	if err != nil {
		err_msg := fmt.Sprintf("Fail to create extract dir [%s]: %s", tmp_extract_dir, err.Error())
		return errors.New(err_msg)
	}

	// tar -xjf ~/.knowbase/knowbase_linux_amd64_1.0.0.tar.bz2 -C ~/.knowbase/tmp/knowbase_linux_amd64_1.0.0
	cmd = fmt.Sprintf("tar -xjf %s -C %s", local_package_path, tmp_extract_dir)
	_, err = utils.RunCommand(cmd, update.OS)
	if err != nil {
		err_msg := fmt.Sprintf("Fail to untar new client package: %s", err.Error())
		return errors.New(err_msg)
	}

	// rm -rf ~/.knowbase/knowbase_linux_amd64_1.0.0.tar.bz2
	os.RemoveAll(local_package_path)

	// 3. backup current package
	knowbase_dir := filepath.Join(update.LocalPackageDir, "knowbase")
	if utils.FileExists(knowbase_dir) {
		// rm -rf knowbase.old
		old_knowbase_dir := filepath.Join(update.LocalPackageDir, "knowbase.old")
		if utils.FileExists(old_knowbase_dir) {
			err = os.RemoveAll(old_knowbase_dir)
			if err != nil {
				err_msg := fmt.Sprintf("Fail to remove [%s]: %s", old_knowbase_dir, err.Error())
				return errors.New(err_msg)
			}
		}

		// mv knowbase to knowbase.old
		cmd = ""
		if flag, _ := utils.IsSymbolLink(knowbase_dir); flag {
			// 如果~/.knowbase/knowbase 存在，并且是个软连，把实际目录 mv 成 knowbase.old
			real_knowbase_dir, _ := os.Readlink(knowbase_dir)
			real_knowbase_dir = filepath.Join(update.LocalPackageDir, real_knowbase_dir)
			if utils.FileExists(real_knowbase_dir) {
				cmd = fmt.Sprintf("mv %s %s", real_knowbase_dir, old_knowbase_dir)
			} else {
				// ~/.knowbase/knowbase 软链接到一个不存在的目录
				os.RemoveAll(real_knowbase_dir)
			}
		} else {
			// 如果~/.knowbase/knowbase 存在，并且不是软连，把该目录 mv 成 knowbase.old
			cmd = fmt.Sprintf("mv %s %s", knowbase_dir, old_knowbase_dir)
		}

		if cmd != "" {
			_, err = utils.RunCommand(cmd, update.OS)
			if err != nil {
				err_msg := fmt.Sprintf("Fail to mv %s to %s: %s", knowbase_dir, old_knowbase_dir, err.Error())
				return errors.New(err_msg)
			}
		}
	}

	// 4. mv ~/.knowbase/tmp/knowbase_linux_amd64_1.0.0  ~/.knowbase/knowbase_linux_amd64_1.0.0
	new_knowbase_dir := filepath.Join(update.LocalPackageDir, new_knowbase)
	if utils.FileExists(new_knowbase_dir) {
		err = os.RemoveAll(new_knowbase_dir)
		if err != nil {
			err_msg := fmt.Sprintf("Fail to remove %s: %s", new_knowbase_dir, err.Error())
			return errors.New(err_msg)
		}
	}
	tmp_knowbase_dir := filepath.Join(tmp_dir, new_knowbase)
	cmd = fmt.Sprintf("mv %s %s", tmp_knowbase_dir, new_knowbase_dir)
	_, err = utils.RunCommand(cmd, update.OS)
	if err != nil {
		err_msg := fmt.Sprintf("Fail to mv [%s] to [%s]: %s", tmp_knowbase_dir, new_knowbase_dir, err.Error())
		return errors.New(err_msg)
	}

	os.RemoveAll(tmp_dir)

	// 4. create symbol link knowbase --> knowbase_linux_amd64_1.0.0
	cmd = fmt.Sprintf("cd %s && ln -snf %s knowbase", update.LocalPackageDir, new_knowbase)
	_, err = utils.RunCommand(cmd, update.OS)
	if err != nil {
		err_msg := fmt.Sprintf("Fail to set knowbase link: %s", err.Error())
		return errors.New(err_msg)
	}

	return nil
}

// get bos root path
// online : http://knowbase-client.bj.bcebos.com/knowbase
// test: http://knowbase-client.bj.bcebos.com/knowbase-test
func (update *Update) GetBosRoot() string {
	// check .test file exist
	testPath := filepath.Join(update.LocalPackageDir, "knowbase", ".test")

	// check file, if not exist, return online bos root
	if _, err := os.Stat(testPath); os.IsNotExist(err) {
		return "http://knowbase-client.bj.bcebos.com/knowbase"
	} else {
		return "http://knowbase-client.bj.bcebos.com/knowbase-test"
	}
}
