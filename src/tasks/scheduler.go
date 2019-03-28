package tasks

import (
	"errors"
	"github.com/robfig/cron"
	log "github.com/sirupsen/logrus"
)

type Scheduler struct {
	tasks   []*Task
	cron    *cron.Cron
	started bool
}

func (s *Scheduler) AddTasks(taskList []*Task) error {
	s.tasks = taskList
	for _, task := range s.tasks {
		if err := s.cron.AddFunc(task.Crontab, task.Run); err != nil {
			return err
		}
	}

	return nil
}

func (s *Scheduler) Start() error {
	if s.started {
		return errors.New("cannot start scheduler, already started")
	}
	s.cron.Start()
	log.Info("scheduler was started")
	s.started = true

	return nil
}

func (s *Scheduler) Stop() error {
	if s.started {
		return errors.New("cannot stop scheduler, already stopped")
	}
	s.cron.Stop()
	log.Info("scheduler was stopped")
	s.started = false

	return nil
}

func NewScheduler() (*Scheduler, error) {
	sched := Scheduler{}
	sched.cron = cron.New()
	sched.started = false

	return &sched, nil
}
