Here’s a complete and professional `README.md` for your **Task Scheduler** package, covering usage, design, features, and setup:

---

````markdown
# 🕒 schedulr — Go Task Scheduler

`schedulr` is a priority-based task scheduler written in Go. It supports one-time and recurring task execution, dynamic worker scaling, timeout handling, and task cancellation — all without a single external dependency.

---

## ✨ Features

- ⏱️ **Priority Queue** — Tasks with higher priority are executed first
- 🧠 **Dynamic Worker Scaling** — Auto-adjusts worker count based on queue size
- ⏳ **Timeout Support** — Cancel tasks that exceed their allowed runtime
- 🔁 **Recurring Tasks** — Schedule tasks to run at fixed intervals
- 📅 **One-Time Scheduling** — Delay execution until a specific time
- ❌ **Graceful Cancellation** — Cancel scheduled tasks by ID
- ✅ **Graceful Shutdown** — Ensures all running tasks complete before exit

---

## 📦 Installation

```bash
go get github.com/kehl-gopher/schedulr
```
````

---

## 🚀 Getting Started

### Initialize the Scheduler

```go
scheduler := schedulr.SchedulerInit()
defer scheduler.ShutDown()
```

---

### Create and Submit a Task

```go
job := func() error {
	fmt.Println("Do something important")
	return nil
}

task := schedulr.NewTask(2*time.Second, job, 5) // timeout = 2s, priority = 5
scheduler.Submit(task)
```

---

### Schedule a One-Time Task

```go
scheduler.ScheduleOnce(job, time.Now().Add(5*time.Second)) // run after 5 seconds
```

---

### Schedule a Recurring Task

```go
id, _ := scheduler.ScheduleRecurring(job, 10*time.Second) // run every 10 seconds

// Cancel it later
scheduler.Cancel(id)
```

---

## 🔒 Task Priority

Tasks are executed in order of **highest priority first**.
If multiple tasks have the same priority, they are executed in the order they were added.

---

## 🧪 Testing

```bash
go test -v ./...
```

Includes:

- ✅ Task execution
- ⏳ Timeout handling
- ⬆️ Priority ordering
- 🔁 Recurring task scheduling
- ❌ Task cancellation

---

## 📐 Architecture

```
+-----------------------------+
|        Task Scheduler       |
+-----------------------------+
| Priority Queue (heap)       |
| Dynamic Worker Pool         |
| Task Timeout Context        |
| Scheduled & Recurring Tasks |
+-----------------------------+
```

---

## 📌 TODO / Improvements

- [ ] Retry failed tasks with backoff
- [ ] Persist task queue to disk
- [ ] Metrics (task count, failures, etc.)
- [ ] Web dashboard for visibility
- [ ] handle running cron task

---

## 🧠 Example Use Cases

- Running background jobs (e.g., emails, billing)
- Queueing delayed tasks (e.g., notifications)
- Real-time task dispatchers with load-based scaling
