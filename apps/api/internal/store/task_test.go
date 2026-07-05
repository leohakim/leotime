package store

import (
	"context"
	"errors"
	"testing"

	"github.com/leotime/leotime/apps/api/internal/db"
)

func TestTaskLifecycle(t *testing.T) {
	ctx := context.Background()
	st, user := newTaskTestStore(t, ctx)

	project, err := st.CreateProject(ctx, user.ID, ProjectInput{
		Name:  "Portal Web",
		Color: "#2563eb",
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	task, err := st.CreateTask(ctx, user.ID, TaskInput{
		ProjectID: project.ID,
		Name:      "  API serializers  ",
		Billable:  true,
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if task.Name != "API serializers" || task.ProjectName != "Portal Web" || !task.Billable {
		t.Fatalf("expected normalized task, got %+v", task)
	}

	tasks, err := st.ListTasks(ctx, user.ID, false, "")
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected one task, got %d", len(tasks))
	}

	updated, err := st.UpdateTask(ctx, user.ID, task.ID, TaskInput{
		Name:     "API serializers updated",
		Billable: false,
	})
	if err != nil {
		t.Fatalf("update task: %v", err)
	}
	if updated.ProjectID != "" || updated.Name != "API serializers updated" || updated.Billable {
		t.Fatalf("unexpected updated task: %+v", updated)
	}

	if err := st.ArchiveTask(ctx, user.ID, task.ID); err != nil {
		t.Fatalf("archive task: %v", err)
	}

	activeTasks, err := st.ListTasks(ctx, user.ID, false, "")
	if err != nil {
		t.Fatalf("list active tasks: %v", err)
	}
	if len(activeTasks) != 0 {
		t.Fatalf("expected no active tasks, got %d", len(activeTasks))
	}

	allTasks, err := st.ListTasks(ctx, user.ID, true, "")
	if err != nil {
		t.Fatalf("list all tasks: %v", err)
	}
	if len(allTasks) != 1 || allTasks[0].ArchivedAt == "" {
		t.Fatalf("expected archived task, got %+v", allTasks)
	}
}

func TestListTasksFiltersByProject(t *testing.T) {
	ctx := context.Background()
	st, user := newTaskTestStore(t, ctx)

	projectA, err := st.CreateProject(ctx, user.ID, ProjectInput{Name: "Project A", Color: "#2563eb"})
	if err != nil {
		t.Fatalf("create project A: %v", err)
	}
	projectB, err := st.CreateProject(ctx, user.ID, ProjectInput{Name: "Project B", Color: "#0f7a5b"})
	if err != nil {
		t.Fatalf("create project B: %v", err)
	}

	if _, err := st.CreateTask(ctx, user.ID, TaskInput{ProjectID: projectA.ID, Name: "Task A", Billable: true}); err != nil {
		t.Fatalf("create task A: %v", err)
	}
	if _, err := st.CreateTask(ctx, user.ID, TaskInput{ProjectID: projectB.ID, Name: "Task B", Billable: true}); err != nil {
		t.Fatalf("create task B: %v", err)
	}

	filtered, err := st.ListTasks(ctx, user.ID, false, projectA.ID)
	if err != nil {
		t.Fatalf("list filtered tasks: %v", err)
	}
	if len(filtered) != 1 || filtered[0].Name != "Task A" {
		t.Fatalf("unexpected filtered tasks: %+v", filtered)
	}
}

func TestCreateTaskValidatesInput(t *testing.T) {
	ctx := context.Background()
	st, user := newTaskTestStore(t, ctx)

	tests := []struct {
		name  string
		input TaskInput
	}{
		{
			name:  "missing name",
			input: TaskInput{Name: "", Billable: true},
		},
		{
			name:  "unknown project",
			input: TaskInput{Name: "Task", ProjectID: "prj_missing", Billable: true},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := st.CreateTask(ctx, user.ID, test.input); !errors.Is(err, ErrInvalidTaskInput) {
				t.Fatalf("expected invalid input, got %v", err)
			}
		})
	}
}

func TestCreateTaskRequiresProjectWhenSettingEnabled(t *testing.T) {
	ctx := context.Background()
	st, user := newTaskTestStore(t, ctx)

	if _, err := st.db.ExecContext(ctx, `
		UPDATE app_settings
		SET task_project_required = 1, updated_at = ?
		WHERE user_id = ?
	`, nowString(), user.ID); err != nil {
		t.Fatalf("enable task project requirement: %v", err)
	}

	if _, err := st.CreateTask(ctx, user.ID, TaskInput{Name: "Standalone task", Billable: true}); !errors.Is(err, ErrInvalidTaskInput) {
		t.Fatalf("expected project required error, got %v", err)
	}
}

func newTaskTestStore(t *testing.T, ctx context.Context) (*Store, *User) {
	t.Helper()

	database, err := db.Open(ctx, t.TempDir()+"/leotime.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		database.Close()
	})

	if err := db.Migrate(ctx, database); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	st := New(database)
	if err := st.BootstrapAdmin(ctx, "admin@example.com", "change-me-now"); err != nil {
		t.Fatalf("bootstrap admin: %v", err)
	}
	user, err := st.Authenticate(ctx, "admin@example.com", "change-me-now")
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	return st, user
}
