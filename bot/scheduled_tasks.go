package bot

import (
	"fmt"
	"sync"

	"github.com/robfig/cron"
)

var taskRunner *cron.Cron
var schedMutex sync.Mutex

func scheduleTasks() {
	schedMutex.Lock()
	if taskRunner != nil {
		taskRunner.Stop()
	}
	botCfg.RLock()
	scheduled := botCfg.ScheduledJobs
	tz := botCfg.timeZone
	botCfg.RUnlock()
	if tz != nil {
		Log(Info, fmt.Sprintf("Scheduling tasks in TimeZone: %s", tz))
		taskRunner = cron.NewWithLocation(tz)
	} else {
		Log(Info, "Scheduling tasks in system default timezone")
		taskRunner = cron.New()
	}
	currentTasks.Lock()
	tasks := taskList{
		currentTasks.t,
		currentTasks.nameMap,
		currentTasks.idMap,
		currentTasks.nameSpaces,
	}
	currentTasks.Unlock()
	confLock.RLock()
	repolist := repositories
	confLock.RUnlock()
	for _, st := range scheduled {
		t := tasks.getTaskByName(st.Name)
		if t == nil {
			Log(Error, fmt.Sprintf("Task not found when scheduling task: %s", st.Name))
			continue
		}
		task, _, job := getTask(t)
		if job == nil {
			Log(Error, fmt.Sprintf("Ignoring '%s' in ScheduledJobs: not a job", st.Name))
			continue
		}
		if task.Disabled {
			Log(Error, fmt.Sprintf("Not scheduling disabled job '%s'; reason: %s", st.Name, task.reason))
			continue
		}
		if len(task.Channel) == 0 {
			Log(Error, fmt.Sprintf("Not scheduling job '%s'; zero-length Channel", st.Name))
			continue
		}
		ts := st.TaskSpec
		Log(Info, fmt.Sprintf("Scheduling job '%s', args '%v' with schedule: %s", ts.Name, ts.Arguments, st.Schedule))
		taskRunner.AddFunc(st.Schedule, func() { runScheduledTask(t, ts, tasks, repolist) })
	}
	taskRunner.Start()
	schedMutex.Unlock()
}

func runScheduledTask(t interface{}, ts TaskSpec, tasks taskList, repolist map[string]repository) {
	task, plugin, _ := getTask(t)
	isPlugin := plugin != nil
	if isPlugin && len(ts.Command) == 0 {
		Log(Error, fmt.Sprintf("Empty 'Command' when running scheduled task '%s' of type plugin", ts.Name))
		return
	}

	botCfg.RLock()
	// Create the botContext to carry state through the pipeline.
	// startPipeline will take care of registerActive()
	c := &botContext{
		Channel:       task.Channel,
		tasks:         tasks,
		repositories:  repolist,
		isCommand:     isPlugin,
		directMsg:     false,
		automaticTask: true, // scheduled jobs don't get authorization / elevation checks
		environment:   make(map[string]string),
	}
	botCfg.RUnlock()
	var command string
	if isPlugin {
		command = ts.Command
	} else {
		command = "run"
	}
	Log(Info, fmt.Sprintf("Starting scheduled task: %s", task.name))
	c.startPipeline(nil, t, scheduled, command, ts.Arguments...)
}
