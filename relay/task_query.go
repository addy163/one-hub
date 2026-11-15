package relay

import (
	"encoding/json"
	"net/http"
	"one-api/model"

	"github.com/gin-gonic/gin"
)

// 通用任务查询协议。new-api 有完整的 /v1/tasks/:key 与 /v1/videos/:id 家族，
// one-hub 原本只能通过 /mj、/suno、/kling 各自的私有路径查任务。
// 这里复用现有 Task 表与轮询器，补一层统一的只读查询入口，不引入新表、新轮询器，
// 也不依赖 JS 插件运行时。

func findTask(c *gin.Context) *model.Task {
	taskID := c.Param("key")
	if taskID == "" {
		taskID = c.Param("task_id")
	}
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "task id is required", "type": "invalid_request_error"}})
		return nil
	}

	task, err := model.GetTaskByTaskIdForUser(c.GetInt("id"), taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "failed to query task", "type": "system_error"}})
		return nil
	}
	if task == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "task not found", "type": "invalid_request_error"}})
		return nil
	}
	return task
}

// GetTaskByKey 返回与平台无关的统一任务状态。
func GetTaskByKey(c *gin.Context) {
	task := findTask(c)
	if task == nil {
		return
	}
	c.JSON(http.StatusOK, unifiedTaskResponse(task))
}

// GetTaskArtifacts 返回任务产物列表（原始 Data 字段原样透出）。
func GetTaskArtifacts(c *gin.Context) {
	task := findTask(c)
	if task == nil {
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"object":  "list",
		"task_id": task.TaskID,
		"data":    rawOrEmpty(task.Data),
	})
}

// GetVideoTaskByID 把内部任务状态映射为 OpenAI /v1/videos/{id} 风格。
func GetVideoTaskByID(c *gin.Context) {
	task := findTask(c)
	if task == nil {
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":         task.TaskID,
		"object":     "video",
		"model":      task.Action,
		"status":     videoTaskStatus(task.Status),
		"progress":   task.Progress,
		"created_at": task.CreatedAt,
		"error":      nullableString(task.FailReason),
		"metadata":   rawOrEmpty(task.Properties),
	})
}

func unifiedTaskResponse(task *model.Task) gin.H {
	return gin.H{
		"task_id":     task.TaskID,
		"platform":    task.Platform,
		"action":      task.Action,
		"status":      string(task.Status),
		"progress":    task.Progress,
		"fail_reason": task.FailReason,
		"created_at":  task.CreatedAt,
		"submit_time": task.SubmitTime,
		"start_time":  task.StartTime,
		"finish_time": task.FinishTime,
		"properties":  rawOrEmpty(task.Properties),
		"data":        rawOrEmpty(task.Data),
	}
}

func videoTaskStatus(status model.TaskStatus) string {
	switch status {
	case model.TaskStatusNotStart, model.TaskStatusSubmitted, model.TaskStatusQueued:
		return "queued"
	case model.TaskStatusInProgress:
		return "in_progress"
	case model.TaskStatusSuccess:
		return "completed"
	case model.TaskStatusFailure, model.TaskStatusUnknown:
		return "failed"
	default:
		return "queued"
	}
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func rawOrEmpty(data []byte) json.RawMessage {
	if len(data) == 0 {
		return json.RawMessage("null")
	}
	if !json.Valid(data) {
		return json.RawMessage("null")
	}
	return json.RawMessage(data)
}
