package task

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Manager handles task file operations and project management
type Manager struct {
	tasksDir string
	mutex    sync.RWMutex
}

// NewManager creates a new task manager
func NewManager(tasksDir string) (*Manager, error) {
	// Create tasks directory if it doesn't exist
	if err := os.MkdirAll(tasksDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create tasks directory: %w", err)
	}

	return &Manager{
		tasksDir: tasksDir,
	}, nil
}

// GetTaskFilePath returns the path to a project's task file
func (m *Manager) GetTaskFilePath(projectName string) string {
	sanitizedName := SanitizeProjectName(projectName)
	return filepath.Join(m.tasksDir, sanitizedName+".md")
}

// GetTasksDir returns the tasks directory path
func (m *Manager) GetTasksDir() string {
	return m.tasksDir
}

// ProjectExists checks if a project file exists
func (m *Manager) ProjectExists(projectName string) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	filePath := m.GetTaskFilePath(projectName)
	_, err := os.Stat(filePath)
	return err == nil
}

// CreateProject creates a new project file
func (m *Manager) CreateProject(projectName string) error {
	if err := ValidateProjectName(projectName); err != nil {
		return err
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	filePath := m.GetTaskFilePath(projectName)

	// Check if file already exists
	if _, err := os.Stat(filePath); err == nil {
		return fmt.Errorf("project file already exists: %s", filePath)
	}

	// Create initial project structure
	project := Project{
		Name:      projectName,
		Tasks:     []Task{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Generate initial markdown content
	content := m.generateMarkdown(project)

	// Write to file
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to create project file: %w", err)
	}

	return nil
}

// LoadProject loads a project from its markdown file
func (m *Manager) LoadProject(projectName string) (*Project, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	filePath := m.GetTaskFilePath(projectName)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("project file not found: %s", projectName)
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read project file: %w", err)
	}

	// Parse markdown content
	project, err := m.parseMarkdown(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse project file: %w", err)
	}

	project.Name = projectName
	return project, nil
}

// SaveProject saves a project to its markdown file
func (m *Manager) SaveProject(project *Project) error {
	if err := ValidateProjectName(project.Name); err != nil {
		return err
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	project.UpdatedAt = time.Now()

	// Generate markdown content
	content := m.generateMarkdown(*project)

	// Write to file
	filePath := m.GetTaskFilePath(project.Name)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to save project file: %w", err)
	}

	return nil
}

// AddTask adds a new task to a project
func (m *Manager) AddTask(projectName string, task Task) error {
	project, err := m.LoadProject(projectName)
	if err != nil {
		return err
	}

	// Set task ID (simple incrementing ID)
	maxID := 0
	for _, existingTask := range project.Tasks {
		if existingTask.ID > maxID {
			maxID = existingTask.ID
		}
	}
	task.ID = maxID + 1
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()

	// Set defaults if not provided
	if task.Status == "" {
		task.Status = DefaultTaskStatus()
	}
	if task.Priority == "" {
		task.Priority = DefaultTaskPriority()
	}

	// Add task to project
	project.Tasks = append(project.Tasks, task)

	// Save project
	return m.SaveProject(project)
}

// UpdateTaskStatus updates the status of a task or subtask
func (m *Manager) UpdateTaskStatus(projectName string, taskTitle string, subtaskTitle string, status TaskStatus) error {
	project, err := m.LoadProject(projectName)
	if err != nil {
		return err
	}

	// Find the task
	taskFound := false
	for i := range project.Tasks {
		if project.Tasks[i].Title == taskTitle {
			taskFound = true

			if subtaskTitle == "" {
				// Update main task status
				if status == StatusDone {
					// When marking a task as done, check if we should auto-complete subtasks
					if len(project.Tasks[i].Subtasks) > 0 {
						// Auto-complete all subtasks when main task is marked done
						for j := range project.Tasks[i].Subtasks {
							if project.Tasks[i].Subtasks[j].Status != StatusDone {
								project.Tasks[i].Subtasks[j].Status = StatusDone
								project.Tasks[i].Subtasks[j].UpdatedAt = time.Now()
							}
						}
					}
				}
				project.Tasks[i].Status = status
				project.Tasks[i].UpdatedAt = time.Now()
			} else {
				// Update subtask status
				subtaskFound := false
				for j := range project.Tasks[i].Subtasks {
					if project.Tasks[i].Subtasks[j].Title == subtaskTitle {
						project.Tasks[i].Subtasks[j].Status = status
						project.Tasks[i].Subtasks[j].UpdatedAt = time.Now()
						project.Tasks[i].UpdatedAt = time.Now()

						// If this was the last subtask to be completed, check if main task should be auto-completed
						if status == StatusDone && project.Tasks[i].Status != StatusDone {
							if project.Tasks[i].CanBeMarkedComplete() {
								project.Tasks[i].Status = StatusDone
								project.Tasks[i].UpdatedAt = time.Now()
							}
						}

						subtaskFound = true
						break
					}
				}
				if !subtaskFound {
					return fmt.Errorf("subtask not found: %s", subtaskTitle)
				}
			}
			break
		}
	}

	if !taskFound {
		return fmt.Errorf("task not found: %s", taskTitle)
	}

	// Save project
	return m.SaveProject(project)
}

// GetNextTask returns the next uncompleted task
func (m *Manager) GetNextTask(projectName string) (*Task, *Subtask, error) {
	project, err := m.LoadProject(projectName)
	if err != nil {
		return nil, nil, err
	}

	// Find first incomplete task/subtask
	for _, task := range project.Tasks {
		// Use IsFullyCompleted to check both task and subtask completion
		if !task.IsFullyCompleted() {
			// Check for incomplete subtasks first
			for _, subtask := range task.Subtasks {
				if subtask.Status != StatusDone {
					return &task, &subtask, nil
				}
			}
			// If no incomplete subtasks but task isn't done, return the main task
			if task.Status != StatusDone {
				return &task, nil, nil
			}
		}
	}

	return nil, nil, fmt.Errorf("all tasks completed")
}

// ListProjects returns a list of all project names
func (m *Manager) ListProjects() ([]string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	files, err := os.ReadDir(m.tasksDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read tasks directory: %w", err)
	}

	var projects []string
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".md" {
			name := strings.TrimSuffix(file.Name(), ".md")
			projects = append(projects, name)
		}
	}

	return projects, nil
}
