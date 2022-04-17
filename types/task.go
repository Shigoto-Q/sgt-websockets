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
	Stats       string = `shigoto-stats`
)

type DockerImage struct {
	Repository string `json:"Repository"`
	Name       string `json:"Name"`
	ImageName  string `json:"ImageName"`
}

type ShigotoStats struct {
	TotalTaskResults           int `json:"totalTaskResults"`
	TotalKubernetesDeployments int `json:"totalKubernetesDeployments"`
	TotalDockerImages          int `json:"totalDockerImages"`
}
