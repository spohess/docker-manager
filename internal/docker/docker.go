package docker

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

func Available() error {
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("comando \"docker\" não encontrado no PATH")
	}
	return nil
}

func ComposeArgs(files []string, args ...string) []string {
	out := make([]string, 0, 1+2*len(files)+len(args))
	out = append(out, "compose")
	for _, file := range files {
		out = append(out, "-f", file)
	}
	return append(out, args...)
}

func ExecArgs(container string, tty bool, command []string) []string {
	mode := "-i"
	if tty {
		mode = "-it"
	}
	return append([]string{"exec", mode, container}, command...)
}

func Interactive() bool {
	return isTerminal(os.Stdin) && isTerminal(os.Stdout)
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func Run(dir string, args []string) error {
	cmd := exec.Command("docker", args...)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s falhou: %w", Display(args), err)
	}
	return nil
}

func Exec(dir string, args []string) error {
	path, err := exec.LookPath("docker")
	if err != nil {
		return fmt.Errorf("comando \"docker\" não encontrado no PATH")
	}
	if dir != "" {
		if err := os.Chdir(dir); err != nil {
			return fmt.Errorf("não foi possível acessar o diretório %s: %w", dir, err)
		}
	}
	if err := syscall.Exec(path, append([]string{"docker"}, args...), os.Environ()); err != nil {
		return fmt.Errorf("não foi possível executar %s: %w", Display(args), err)
	}
	return nil
}

func Display(args []string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, "docker")
	for _, arg := range args {
		if arg == "" || strings.ContainsAny(arg, " \t\"'") {
			arg = fmt.Sprintf("%q", arg)
		}
		parts = append(parts, arg)
	}
	return strings.Join(parts, " ")
}
