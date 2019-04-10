package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"

	"github.com/joho/godotenv"
)

type envVars []string

func (l *envVars) Set(v string) error {
	*l = append(*l, v)
	return nil
}

func (l *envVars) String() string {
	return fmt.Sprintf("%v", []string(*l))
}

func run(in io.Reader, out io.Writer, errOut io.Writer, args []string) int {
	flags := flag.NewFlagSet(filepath.Base(args[0]), flag.ExitOnError)
	var (
		env       = &envVars{}
		useDotenv bool
		listEnvs  bool
	)
	flags.Var(env, "e", "Environment variables with `NAME=VALUE` form")
	flags.BoolVar(&useDotenv, "r", false, "Read enviroment variables from .env")
	flags.BoolVar(&listEnvs, "l", false, "List environment variables")
	if err := flags.Parse(args[1:]); err != nil {
		log.Printf("Argument error: %s", err)
		return 128
	} else if flags.NArg() <= 0 {
		log.Printf("No positional arguments")
		return 128
	}

	if useDotenv {
		if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
			log.Printf("Unable to load .env file: %s", err)
			return 1
		}
	}

	if listEnvs {
		for _, envVar := range os.Environ() {
			fmt.Fprintln(out, envVar)
		}
		return 0
	}

	cmdName := flags.Args()[0]
	var cmdArgs []string
	if flags.NArg() > 1 {
		cmdArgs = flags.Args()[1:]
	}
	cmd := exec.Command(cmdName, cmdArgs...)
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = errOut
	cmd.Env = append(os.Environ(), (*env)...)
	if err := cmd.Start(); err != nil {
		log.Print(err)
		return 1
	}
	sigCh := make(chan os.Signal)
	signal.Notify(sigCh)
	defer func() {
		if !cmd.ProcessState.Exited() {
			cmd.Process.Kill()
		}
	}()
	go func() {
		defer signal.Stop(sigCh)
		for sig := range sigCh {
			cmd.Process.Signal(sig)
		}
	}()
	if err := cmd.Wait(); err != nil {
		log.Print(err)
		return 1
	}
	return 0
}

func main() {
	os.Exit(run(os.Stdin, os.Stdout, os.Stderr, os.Args))
}
