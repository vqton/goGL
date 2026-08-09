package task

import (
	"context"

	"goGL/internal/domain/core"
)

type Task struct {
	ID       string      `json:"id" bson:"_id"`
	Title    string      `json:"title" bson:"title"`
	DueDate  string      `json:"due_date" bson:"due_date"`
	Assignee string      `json:"assignee" bson:"assignee"`
	Status   core.Status `json:"status" bson:"status"`
}

type Repository interface {
	Create(ctx context.Context, t *Task) error
	FindByID(ctx context.Context, id string) (*Task, error)
	Update(ctx context.Context, t *Task) error
}
