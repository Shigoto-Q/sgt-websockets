package types

type TaskResult struct {
	TaskId     string `json:"taskId"`
	TaskName   string `json:"taskName"`
	Status     int    `json:"status"`
	User       string `json:"user"`
	UserId     int    `json:"userId"`
	FinishedAt string `json:"finishedAt"`
}

type TaskCountMessage struct {
	Success int `json:"success"`
	Failure int `json:"failure"`
	Pending int `json:"pending"`
	UserId  int `json:"userId"`
}

const (
	TaskCount   string = "task_count"
	TaskResults string = "task_results"
)

type DockerImage struct {
	Repository string `json:"Repository"`
	Name       string `json:"Name"`
	ImageName  string `json:"ImageName"`
}
