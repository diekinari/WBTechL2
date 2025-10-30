// minishell.go
package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// глобальная переменная для текущей foreground PGID (0 == none)
var fgPgID int

func main() {
	// перехватываем SIGINT (Ctrl+C)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		for range sigCh {
			// если есть foreground группа — посылаем SIGINT ей
			if fgPgID != 0 {
				// send SIGINT to process group (-pgid)
				_ = syscall.Kill(-fgPgID, syscall.SIGINT)
			} else {
				// печатаем новую строку тк пользователь нажал Ctrl+C в shell
				fmt.Println()
			}
		}
	}()

	reader := bufio.NewReader(os.Stdin)
	for {
		wd, _ := os.Getwd()
		fmt.Printf("%s $ ", filepath.Base(wd))
		line, err := readLine(reader)
		if err != nil {
			if err == io.EOF {
				fmt.Println("\nexit")
				return
			}
			fmt.Fprintln(os.Stderr, "read error:", err)
			continue
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// expand env vars
		line = os.ExpandEnv(line)

		// process conditionals (&& and ||)
		segments, ops := splitByConditionals(line)
		var lastExit int = 0
		var lastErr error = nil
		for i, seg := range segments {
			// determine if we should run this segment
			if i > 0 {
				if ops[i-1] == "&&" && lastExit != 0 {
					// skip
					continue
				}
				if ops[i-1] == "||" && lastExit == 0 {
					// skip
					continue
				}
			}
			exit, err := runSegment(strings.TrimSpace(seg))
			lastExit = exit
			lastErr = err
			// continue to next segment depending on ops handled above
		}
		// optionally could print lastErr
		_ = lastErr
	}
}

// readLine reads a line, returns io.EOF when Ctrl+D
func readLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		// if EOF and have partial data, return it
		if err == io.EOF {
			if len(line) == 0 {
				return "", io.EOF
			}
			// else return partial line and nil
			return strings.TrimRight(line, "\n"), nil
		}
		return "", err
	}
	return strings.TrimRight(line, "\n"), nil
}

// splitByConditionals splits input into segments separated by && or || (not inside quotes).
// returns segments and list of operators in-between
func splitByConditionals(s string) (segments []string, ops []string) {
	var cur bytes.Buffer
	inSq := false
	inDq := false
	for i := 0; i < len(s); i++ {
		if s[i] == '\'' && !inDq {
			inSq = !inSq
			cur.WriteByte(s[i])
			continue
		}
		if s[i] == '"' && !inSq {
			inDq = !inDq
			cur.WriteByte(s[i])
			continue
		}
		// check for && or ||
		if !inSq && !inDq && i+1 < len(s) {
			if s[i] == '&' && s[i+1] == '&' {
				segments = append(segments, cur.String())
				cur.Reset()
				ops = append(ops, "&&")
				i++
				continue
			}
			if s[i] == '|' && s[i+1] == '|' {
				segments = append(segments, cur.String())
				cur.Reset()
				ops = append(ops, "||")
				i++
				continue
			}
		}
		cur.WriteByte(s[i])
	}
	segments = append(segments, cur.String())
	return
}

// runSegment executes a single segment, which may contain pipelines (|).
// returns exit code and error
func runSegment(segment string) (int, error) {
	// Split by pipes respecting quotes
	parts := splitByPipe(segment)
	// if single command and builtin -> handle builtin
	if len(parts) == 1 {
		args, inFile, outFile, err := parseCommand(parts[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, "parse error:", err)
			return 1, err
		}
		if len(args) == 0 {
			return 0, nil
		}
		// check builtin
		switch args[0] {
		case "cd", "pwd", "echo", "kill", "ps":
			return runBuiltin(args, inFile, outFile)
		}
	}

	// Otherwise run pipeline of external commands (and possibly builtins as external)
	return runPipeline(parts)
}

// splitByPipe - splits by '|' not inside quotes
func splitByPipe(s string) []string {
	var res []string
	var cur bytes.Buffer
	inSq := false
	inDq := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\'' && !inDq {
			inSq = !inSq
			cur.WriteByte(c)
			continue
		}
		if c == '"' && !inSq {
			inDq = !inDq
			cur.WriteByte(c)
			continue
		}
		if c == '|' && !inSq && !inDq {
			res = append(res, strings.TrimSpace(cur.String()))
			cur.Reset()
			continue
		}
		cur.WriteByte(c)
	}
	res = append(res, strings.TrimSpace(cur.String()))
	return res
}

// parseCommand parses a single command (no pipes) into args and redirections.
// Returns args slice (with quotes removed), input file (or ""), output file (or ""), error.
func parseCommand(cmd string) (args []string, inFile string, outFile string, err error) {
	toks, err := splitArgs(cmd)
	if err != nil {
		return nil, "", "", err
	}
	var clean []string
	for i := 0; i < len(toks); i++ {
		t := toks[i]
		if t == ">" {
			if i+1 >= len(toks) {
				return nil, "", "", errors.New("no filename after >")
			}
			outFile = toks[i+1]
			i++
			continue
		}
		if t == "<" {
			if i+1 >= len(toks) {
				return nil, "", "", errors.New("no filename after <")
			}
			inFile = toks[i+1]
			i++
			continue
		}
		clean = append(clean, t)
	}
	return clean, inFile, outFile, nil
}

// splitArgs splits a command string into arguments, respecting single/double quotes.
// Quotes are removed from resulting args.
func splitArgs(s string) ([]string, error) {
	var res []string
	var cur bytes.Buffer
	inSq := false
	inDq := false
	esc := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if esc {
			cur.WriteByte(c)
			esc = false
			continue
		}
		if c == '\\' {
			esc = true
			continue
		}
		if c == '\'' && !inDq {
			inSq = !inSq
			continue // strip quote
		}
		if c == '"' && !inSq {
			inDq = !inDq
			continue // strip quote
		}
		if (c == ' ' || c == '\t') && !inSq && !inDq {
			if cur.Len() > 0 {
				res = append(res, cur.String())
				cur.Reset()
			}
			continue
		}
		cur.WriteByte(c)
	}
	if inSq || inDq {
		return nil, errors.New("unclosed quote")
	}
	if cur.Len() > 0 {
		res = append(res, cur.String())
	}
	return res, nil
}

// runBuiltin runs builtin cmd when pipeline length == 1.
// returns exit code and error.
func runBuiltin(args []string, inFile, outFile string) (int, error) {
	// setup io
	var stdin io.ReadCloser
	var stdout io.WriteCloser
	var err error
	stdin = nil
	stdout = nil

	// input
	if inFile != "" {
		f, e := os.Open(inFile)
		if e != nil {
			fmt.Fprintln(os.Stderr, "open input:", e)
			return 1, e
		}
		defer f.Close()
		stdin = f
	}

	// output
	if outFile != "" {
		f, e := os.Create(outFile)
		if e != nil {
			fmt.Fprintln(os.Stderr, "open output:", e)
			return 1, e
		}
		defer f.Close()
		stdout = f
	}

	// Use provided stdio or fallback to os.Stdin/os.Stdout
	var in io.Reader = os.Stdin
	var out io.Writer = os.Stdout
	if stdin != nil {
		in = stdin
	}
	if stdout != nil {
		out = stdout
	}

	switch args[0] {
	case "cd":
		if len(args) < 2 {
			home := os.Getenv("HOME")
			if home == "" {
				home = "/"
			}
			err = os.Chdir(home)
		} else {
			err = os.Chdir(args[1])
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "cd:", err)
			return 1, err
		}
		return 0, nil
	case "pwd":
		wd, e := os.Getwd()
		if e != nil {
			fmt.Fprintln(os.Stderr, "pwd:", e)
			return 1, e
		}
		fmt.Fprintln(out, wd)
		return 0, nil
	case "echo":
		fmt.Fprintln(out, strings.Join(args[1:], " "))
		return 0, nil
	case "kill":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "kill: pid required")
			return 1, errors.New("pid required")
		}
		pid, e := strconv.Atoi(args[1])
		if e != nil {
			fmt.Fprintln(os.Stderr, "kill: bad pid")
			return 1, e
		}
		// send SIGTERM
		if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
			fmt.Fprintln(os.Stderr, "kill:", err)
			return 1, err
		}
		return 0, nil
	case "ps":
		// run external ps command for portability
		cmd := exec.Command("ps", "aux")
		cmd.Stdout = out
		cmd.Stderr = os.Stderr
		if in != nil {
			cmd.Stdin = in
		}
		if err := cmd.Run(); err != nil {
			fmt.Fprintln(os.Stderr, "ps:", err)
			return 1, err
		}
		return 0, nil
	default:
		return 127, errors.New("unknown builtin")
	}
}

// runPipeline runs a pipeline of commands (each item in parts is a command string).
// returns exit code of the last command and error
func runPipeline(parts []string) (int, error) {
	n := len(parts)
	cmds := make([]*exec.Cmd, n)

	// prepare parsed commands
	type cmdSpec struct {
		args    []string
		inFile  string
		outFile string
	}
	specs := make([]cmdSpec, n)
	for i := 0; i < n; i++ {
		a, inF, outF, err := parseCommand(parts[i])
		if err != nil {
			fmt.Fprintln(os.Stderr, "parse error:", err)
			return 1, err
		}
		specs[i] = cmdSpec{args: a, inFile: inF, outFile: outF}
		if len(a) == 0 {
			return 0, nil
		}
	}

	// create pipes between commands
	pipeReaders := make([]*os.File, n-1)
	pipeWriters := make([]*os.File, n-1)
	for i := 0; i < n-1; i++ {
		r, w, err := os.Pipe()
		if err != nil {
			return 1, err
		}
		pipeReaders[i] = r
		pipeWriters[i] = w
	}

	// Will store started processes to wait for them later
	var procs []*exec.Cmd

	var firstPid int = 0

	// start first command
	for i := 0; i < n; i++ {
		spec := specs[i]
		// if args empty skip
		if len(spec.args) == 0 {
			continue
		}
		// create command
		cmd := exec.Command(spec.args[0], spec.args[1:]...)
		// set stdin
		if i == 0 {
			// first command stdin: either file or os.Stdin
			if spec.inFile != "" {
				f, err := os.Open(spec.inFile)
				if err != nil {
					fmt.Fprintln(os.Stderr, "open input:", err)
					return 1, err
				}
				defer f.Close()
				cmd.Stdin = f
			} else {
				cmd.Stdin = os.Stdin
			}
		} else {
			// stdin from previous pipe
			cmd.Stdin = pipeReaders[i-1]
		}
		// set stdout
		if i == n-1 {
			// last command stdout: either file or os.Stdout
			if spec.outFile != "" {
				f, err := os.Create(spec.outFile)
				if err != nil {
					fmt.Fprintln(os.Stderr, "create output:", err)
					return 1, err
				}
				defer f.Close()
				cmd.Stdout = f
			} else {
				cmd.Stdout = os.Stdout
			}
		} else {
			cmd.Stdout = pipeWriters[i]
		}
		cmd.Stderr = os.Stderr

		// setup process group attributes
		// For first process: setpgid true -> pgid = pid
		// For others: set pgid to firstPid (after first started)
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true,
		}
		// store cmd
		cmds[i] = cmd

		// Start first cmd immediately to obtain pid (so we can set pgid for others)
		if i == 0 {
			if err := cmd.Start(); err != nil {
				fmt.Fprintln(os.Stderr, "start:", err)
				return 1, err
			}
			firstPid = cmd.Process.Pid
			// ensure first process group is its pid (it is by Setpgid:true)
			procs = append(procs, cmd)
			// close writer end in parent if piped
			if n > 1 {
				if pipeWriters[0] != nil {
					_ = pipeWriters[0].Close()
				}
			}
			continue
		}

		// for i > 0: set desired pgid to firstPid
		cmd.SysProcAttr.Pgid = firstPid

		// Now start
		if err := cmd.Start(); err != nil {
			fmt.Fprintln(os.Stderr, "start:", err)
			return 1, err
		}
		procs = append(procs, cmd)

		// close appropriate pipe writer in parent
		if i < n-1 {
			if pipeWriters[i] != nil {
				_ = pipeWriters[i].Close()
			}
		}
		// close previous reader in parent? parent uses none
	}

	// close all pipe readers in parent (they are used by child processes)
	for _, r := range pipeReaders {
		if r != nil {
			// parent can close its copy
			_ = r.Close()
		}
	}
	for _, w := range pipeWriters {
		if w != nil {
			_ = w.Close()
		}
	}

	// Set fgPgID to firstPid so signal handler can forward SIGINT
	fgPgID = firstPid
	// Wait for processes - pipeline exit status is status of last process
	lastExit := 0
	var lastErr error
	for i, p := range procs {
		err := p.Wait()
		if err != nil {
			// try to extract exit status
			if exitErr, ok := err.(*exec.ExitError); ok {
				if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
					lastExit = status.ExitStatus()
				} else {
					lastExit = 1
				}
			} else {
				lastExit = 1
			}
			lastErr = err
		} else {
			// success
			lastExit = 0
			lastErr = nil
		}
		// if this is the last proc, break? we still wait for all to not leave zombies
		_ = i
	}
	// clear fg
	fgPgID = 0
	return lastExit, lastErr
}
