package tasks

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

type taskExecution struct {
	id   int
	time time.Time
	cmd  *exec.Cmd
	stop chan error
}

func (te *taskExecution) Wait() {
	te.stop <- te.cmd.Wait()
}

func (te *taskExecution) Stop() {
	_ = te.cmd.Process.Kill()
	te.stop <- errors.New("interrupted")
}

func newExecution(id int, cmd *exec.Cmd) *taskExecution {
	return &taskExecution{
		id:   id,
		time: time.Now(),
		cmd:  cmd,
		stop: make(chan error, 1),
	}
}

type Task struct {
	Name              string   `yaml:"name"`
	Crontab           string   `yaml:"crontab"`
	Dir               string   `yaml:"dir"`
	UseShell          bool     `yaml:"useShell"`
	Command           string   `yaml:"cmd"`
	Args              []string `yaml:"args"`
	Deadline          uint32   `yaml:"deadline"`
	UseSystemEnv      *bool    `yaml:"useSystemEnv,omitempty"`
	ConcurrencyPolicy string   `yaml:"concurrencyPolicy"`

	// internal values
	envVars []string
	cmdExe  string
	cmdArg  []string
	buf     bytes.Buffer
	writer  io.Writer

	// task executions
	execCount int
	execMap   map[int]*taskExecution
}

func (t *Task) init(shell string, systemEnvs *map[string]string) error {
	t.execCount = 0

	if t.UseSystemEnv == nil {
		b := true
		t.UseSystemEnv = &b
	}
	if *t.UseSystemEnv == true {
		for k, v := range *systemEnvs {
			t.envVars = append(t.envVars, fmt.Sprintf("%s=%s", k, v))
		}
	} else {
		t.envVars = append(t.envVars, fmt.Sprintf("PATH=%s", (*systemEnvs)["PATH"]))
	}
	if shell != "" && t.UseShell {
		t.cmdExe = shell
		t.cmdArg = []string{"-c", t.Command + " " + strings.Join(t.Args, " ")}
	} else {
		t.cmdExe = t.Command
		t.cmdArg = t.Args
	}

	if t.ConcurrencyPolicy == "" {
		t.ConcurrencyPolicy = "Allow"
	} else if t.ConcurrencyPolicy != "Allow" && t.ConcurrencyPolicy != "Forbid" && t.ConcurrencyPolicy != "Replace" {
		return errors.New(fmt.Sprintf("invalid value '%s' for concurrencyPolicy, allowed: Allow, Forbid, Replace", t.ConcurrencyPolicy))
	}

	t.execMap = make(map[int]*taskExecution, 0)

	return nil
}

func (t *Task) getCmd() *exec.Cmd {
	cmd := exec.Command(t.cmdExe, t.cmdArg...)
	if t.Dir != "" {
		cmd.Dir = t.Dir
	}
	cmd.Env = t.envVars

	// setup out pipe
	t.buf.Reset()
	t.writer = bufio.NewWriter(&t.buf)
	cmd.Stdout = t.writer
	cmd.Stderr = t.writer

	return cmd
}

func (t *Task) Run() {
	// work with running executions
	runningTaskCount := len(t.execMap)
	if runningTaskCount > 0 {
		if t.ConcurrencyPolicy == "Forbid" {
			log.Errorf("[%-20s][%d] CANNOT RUN, another %d running executions", t.Name, t.execCount+1, runningTaskCount)
			return
		} else if t.ConcurrencyPolicy == "Replace" {
			log.Infof("[%-20s][%d] found %d running executions, cleaning...", t.Name, t.execCount+1, runningTaskCount)
			for _, ex := range t.execMap {
				ex.Stop()
			}
			log.Infof("[%-20s][%d] %d running executions was cleaned", t.Name, t.execCount+1, runningTaskCount)
		}
	}
	t.execCount++

	// create execution
	execution := newExecution(t.execCount, t.getCmd())
	// start command
	if err := execution.cmd.Start(); err != nil {
		log.Errorf("[%-20s][%d] cannot start: %v", t.Name, execution.id, err)
		return
	}
	// wait for execution ends
	go execution.Wait()
	// add execution to list
	t.execMap[execution.id] = execution

	log.Infof("[%-20s][%d] started", t.Name, execution.id)
	select {
	case err := <-execution.stop:
		if err == nil {
			log.Infof("[%-20s][%d] SUCCESS", t.Name, execution.id)
			log.Debugf("[%-20s][%d] SUCCESS, output: %s", t.Name, execution.id, t.buf.String())
		} else {
			log.Infof("[%-20s][%d] ERROR, err: %v, output: %s", t.Name, execution.id, err, t.buf.String())
		}
		delete(t.execMap, execution.id)
	}
}

// ConfigDescriptiveInfo
type ConfigDescriptiveInfo struct {
	Shell string  `yaml:"shell"`
	Tasks []*Task `yaml:"tasks"`
}

func (di *ConfigDescriptiveInfo) InitTasks() []error {
	var errList = make([]error, 0)
	var err error

	// get os PATH env
	env := make(map[string]string, 0)
	for _, e := range os.Environ() {
		pair := strings.Split(e, "=")
		env[pair[0]] = pair[1]
	}

	var taskNames []string
	for _, t := range di.Tasks {
		if err = t.init(di.Shell, &env); err != nil {
			errList = append(errList, err)
		}
		taskNames = append(taskNames, t.Name)
	}

	log.Infof("%d tasks loaded - %s", len(di.Tasks), strings.Join(taskNames, ", "))

	return errList
}
