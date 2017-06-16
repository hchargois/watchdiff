package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	flags "github.com/hchargois/go-flags"
	"github.com/shopspring/decimal"
)

type Runnable interface {
	Run() string
}

type command struct {
	cmd   string
	args  []string
	shell bool
}

func (c *command) Run() string {
	cmd := exec.Command(c.cmd, c.args...)
	out, _ := cmd.CombinedOutput()
	return string(out)
}

func NewExecCommand(cmd []string) Runnable {
	return &command{
		cmd:   cmd[0],
		args:  cmd[1:],
		shell: false,
	}
}

func NewShellCommand(cmd []string) Runnable {
	args := []string{"-c"}
	cmdstring := strings.Join(cmd, " ")
	args = append(args, cmdstring)
	return &command{
		cmd:   "sh",
		args:  args,
		shell: true,
	}
}

type Options struct {
	Interval float64 `short:"n" long:"interval" description:"Specify update interval in seconds, may have a fractional part, min. 0.1" default:"2"`
	Exec     bool    `short:"x" long:"exec" description:"if not specified, the command is run with 'sh -c'; if specified it is executed directly"`
	Args     struct {
		Command []string `required:"1" positional-arg-name:"command"`
	} `positional-args:"yes" required:"yes"`
}

type looper struct {
	cmd         Runnable
	interval    time.Duration
	initialized bool
	last        decimal.Decimal
	re          *regexp.Regexp
}

func (l *looper) printDiff(out string, actualInterval time.Duration) {
	firstLine := strings.SplitN(out, "\n", 2)[0]
	number := l.re.FindString(firstLine)
	if number == "" {
		fmt.Println(firstLine)
		return
	}
	prev := l.last
	l.last, _ = decimal.NewFromString(number)
	if !l.initialized {
		l.initialized = true
		fmt.Println(firstLine)
		return
	}
	diff := l.last.Sub(prev)
	diffF, _ := diff.Float64()
	diffSec := diffF / (float64(actualInterval) / float64(time.Second))
	fmt.Printf("%v (diff=%v, diff/s=%.2f)\n", firstLine, diff, diffSec)
}

func (l *looper) loop() {
	l.re, _ = regexp.Compile(`\d+(?:\.\d+)?`)
	var startedAt, finishedAt time.Time
	for {
		lastFinishedAt := finishedAt
		startedAt = time.Now()
		out := l.cmd.Run()
		finishedAt = time.Now()
		took := finishedAt.Sub(startedAt)
		actualInterval := finishedAt.Sub(lastFinishedAt)
		l.printDiff(out, actualInterval)
		if took < l.interval {
			time.Sleep(l.interval - took)
		}
	}
}

func main() {
	var opts Options
	parser := flags.NewParser(&opts, flags.Default|flags.PassAfterNonOption)
	_, err := parser.Parse()
	if err != nil {
		os.Exit(1)
	}
	var cmd Runnable
	if opts.Exec {
		cmd = NewExecCommand(opts.Args.Command)
	} else {
		cmd = NewShellCommand(opts.Args.Command)
	}

	if opts.Interval < 0.1 {
		opts.Interval = 0.1
	}

	lpr := &looper{
		cmd:      cmd,
		interval: time.Duration(int64(opts.Interval * float64(time.Second))),
	}
	lpr.loop()
}
