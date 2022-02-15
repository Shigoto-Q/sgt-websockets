package types

type TaskResult struct {
	Task_id      string `json:"task_id"`
	Task_name    string `json:"task_ame"`
	Task_kwargs  string `json:"task_kwargs"`
	Status       string `json:"status"`
	Result       string `json:"result"`
	Traceback    string `json:"traceback"`
	Exception    string `json:"exception"`
	Date_done    string `json:"date_done"`
	Date_created string `json:"date_created"`
	User         string `json:"user"`
	User_id      int    `json:"user_id"`
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
