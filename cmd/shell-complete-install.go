// Copyright (c) 2015-2022 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
)

const shellCompletionMarker = "# mc shell completion"

// bashCompletionScript is adapted from github.com/urfave/cli/v2/autocomplete/bash_autocomplete.
const bashCompletionScript = `#! /bin/bash

: ${PROG:=` + "`basename ${BASH_SOURCE}`" + `}

_cli_init_completion() {
  COMPREPLY=()
  _get_comp_words_by_ref "$@" cur prev words cword
}

_cli_bash_autocomplete() {
  if [[ "${COMP_WORDS[0]}" != "source" ]]; then
    local cur opts base words
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    if declare -F _init_completion >/dev/null 2>&1; then
      _init_completion -n "=:" || return
    else
      _cli_init_completion -n "=:" || return
    fi
    words=("${words[@]:0:$cword}")
    if [[ "$cur" == "-"* ]]; then
      requestComp="${words[*]} ${cur} --generate-bash-completion"
    else
      requestComp="${words[*]} --generate-bash-completion"
    fi
    opts=$(eval "${requestComp}" 2>/dev/null)
    COMPREPLY=($(compgen -W "${opts}" -- ${cur}))
    return 0
  fi
}

complete -o bashdefault -o default -o nospace -F _cli_bash_autocomplete $PROG
`

// zshCompletionScript is adapted from github.com/urfave/cli/v2/autocomplete/zsh_autocomplete.
const zshCompletionScript = `#compdef $PROG

_cli_zsh_autocomplete() {
  local -a opts
  local cur
  cur=${words[-1]}
  if [[ "$cur" == "-"* ]]; then
    opts=("${(@f)$(${words[@]:0:#words[@]-1} ${cur} --generate-bash-completion)}")
  else
    opts=("${(@f)$(${words[@]:0:#words[@]-1} --generate-bash-completion)}")
  fi

  if [[ "${opts[1]}" != "" ]]; then
    _describe 'values' opts
  else
    _files
  fi
}

compdef _cli_zsh_autocomplete $PROG
`

const fishCompletionScript = `# mc shell completion
function __complete_%[1]s
    set -l tokens (commandline -opc)
    set -l current (commandline -ct)
    if test -n "$current"
        set tokens $tokens $current
    end
    %[2]s $tokens --generate-bash-completion
end
complete -f -c %[1]s -a "(__complete_%[1]s)"
`

type shellInstaller interface {
	IsInstalled(cmd, bin string) bool
	Install(cmd, bin string) error
}

type bashInstaller struct {
	rc string
}

func (b bashInstaller) IsInstalled(cmd, _ string) bool {
	return lineInFile(b.rc, shellCompletionMarker) && lineInFile(b.rc, cmd)
}

func (b bashInstaller) Install(cmd, bin string) error {
	if b.IsInstalled(cmd, bin) {
		return fmt.Errorf("already installed in %s", b.rc)
	}
	scriptPath, err := writeCompletionScript(cmd, "bash", bashCompletionScript)
	if err != nil {
		return err
	}
	block := fmt.Sprintf("%s\nPROG=%s\nsource %q\n", shellCompletionMarker, cmd, scriptPath)
	return appendFile(b.rc, block)
}

type zshInstaller struct {
	rc string
}

func (z zshInstaller) IsInstalled(cmd, _ string) bool {
	return lineInFile(z.rc, shellCompletionMarker) && lineInFile(z.rc, cmd)
}

func (z zshInstaller) Install(cmd, bin string) error {
	if z.IsInstalled(cmd, bin) {
		return fmt.Errorf("already installed in %s", z.rc)
	}
	scriptPath, err := writeCompletionScript(cmd, "zsh", zshCompletionScript)
	if err != nil {
		return err
	}
	block := fmt.Sprintf("%s\nPROG=%s\nsource %q\n", shellCompletionMarker, cmd, scriptPath)
	return appendFile(z.rc, block)
}

type fishInstaller struct {
	configDir string
}

func (f fishInstaller) IsInstalled(cmd, _ string) bool {
	completionFile := f.completionFile(cmd)
	if _, err := os.Stat(completionFile); err == nil {
		return true
	}
	return false
}

func (f fishInstaller) Install(cmd, bin string) error {
	if f.IsInstalled(cmd, bin) {
		return fmt.Errorf("already installed at %s", f.completionFile(cmd))
	}
	content := fmt.Sprintf(fishCompletionScript, cmd, bin)
	return writeCompletionFile(f.completionFile(cmd), content)
}

func (f fishInstaller) completionFile(cmd string) string {
	return filepath.Join(f.configDir, "completions", cmd+".fish")
}

func writeCompletionScript(cmd, shell, content string) (string, error) {
	dir, err := completionConfigDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, cmd+"."+shell)
	if err := writeCompletionFile(path, content); err != nil {
		return "", err
	}
	return path, nil
}

func completionConfigDir() (string, error) {
	base, err := configHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "mc", "completions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func configHomeDir() (string, error) {
	if configHome := os.Getenv("XDG_CONFIG_HOME"); configHome != "" {
		return configHome, nil
	}
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return filepath.Join(u.HomeDir, ".config"), nil
}

func shellInstallers() []shellInstaller {
	var installers []shellInstaller
	switch runtime.GOOS {
	case "darwin":
		if f := rcFile(".bash_profile"); f != "" {
			installers = append(installers, bashInstaller{rc: f})
		}
	default:
		for _, name := range []string{".bashrc", ".bash_profile", ".bash_login", ".profile"} {
			if f := rcFile(name); f != "" {
				installers = append(installers, bashInstaller{rc: f})
				break
			}
		}
	}
	if f := rcFile(".zshrc"); f != "" {
		installers = append(installers, zshInstaller{rc: f})
	}
	if d := fishConfigDir(); d != "" {
		installers = append(installers, fishInstaller{configDir: d})
	}
	return installers
}

func fishConfigDir() string {
	configDir, err := configHomeDir()
	if err != nil {
		return ""
	}
	configDir = filepath.Join(configDir, "fish")
	if info, err := os.Stat(configDir); err != nil || !info.IsDir() {
		return ""
	}
	return configDir
}

func rcFile(name string) string {
	u, err := user.Current()
	if err != nil {
		return ""
	}
	path := filepath.Join(u.HomeDir, name)
	if _, err := os.Stat(path); err != nil {
		return ""
	}
	return path
}

func lineInFile(path, needle string) bool {
	content, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(content), needle)
}

func appendFile(path, content string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteString("\n" + content + "\n"); err != nil {
		return err
	}
	return nil
}

func writeCompletionFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func installShellCompletion(cmd string) error {
	installers := shellInstallers()
	if len(installers) == 0 {
		return errors.New("did not find any shells to install")
	}
	bin, err := os.Executable()
	if err != nil {
		return err
	}
	bin, err = filepath.Abs(bin)
	if err != nil {
		return err
	}

	var errOut error
	for _, installer := range installers {
		if err := installer.Install(cmd, bin); err != nil {
			errOut = errors.Join(errOut, err)
		}
	}
	return errOut
}

func isShellCompletionInstalled(cmd string) bool {
	bin, err := os.Executable()
	if err != nil {
		return false
	}
	bin, err = filepath.Abs(bin)
	if err != nil {
		return false
	}
	for _, installer := range shellInstallers() {
		if installer.IsInstalled(cmd, bin) {
			return true
		}
	}
	return false
}
