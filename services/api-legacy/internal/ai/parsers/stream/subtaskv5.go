package streamparser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"semo-server/internal/ai/models"
	"semo-server/internal/ai/streaming"
)

// SubtaskV5State defines state constants specifically for the V5 parser
type SubtaskV5State struct {
	Normal        int
	InFinalAnswer int
	InTodo        int
	InSubTodos    int
	InTask        int
	InObjective   int
	InDeliverable int
}

// V5State contains the state constants for the SubtaskV5StreamParser
var V5State = SubtaskV5State{
	Normal:        0,
	InFinalAnswer: 1,
	InTodo:        2,
	InSubTodos:    3,
	InTask:        4,
	InObjective:   5,
	InDeliverable: 6,
}

// SubtaskV5StreamParser processes streaming subtask V5 output
type SubtaskV5StreamParser struct {
	buffer             strings.Builder
	currentState       int
	currentTaskNumber  int
	currentTaskTitle   string
	currentObjective   strings.Builder
	currentDeliverable strings.Builder
	mainTodo           string
	eventSender        *streaming.EventSender
	mu                 sync.Mutex
	sentObjective      bool
	sentDeliverable    bool
}

// NewSubtaskV5StreamParser creates a new SubtaskV5StreamParser
func NewSubtaskV5StreamParser(streamChan chan<- string) *SubtaskV5StreamParser {
	return &SubtaskV5StreamParser{
		buffer:             strings.Builder{},
		currentState:       V5State.Normal,
		currentTaskNumber:  0,
		currentTaskTitle:   "",
		currentObjective:   strings.Builder{},
		currentDeliverable: strings.Builder{},
		mainTodo:           "",
		eventSender:        streaming.NewEventSender(streamChan),
		mu:                 sync.Mutex{},
		sentObjective:      false,
		sentDeliverable:    false,
	}
}

// GetType returns the type of parser
func (p *SubtaskV5StreamParser) GetType() string {
	return "subtask_v5_stream_parser"
}

// ProcessChunk processes a chunk of streaming output
func (p *SubtaskV5StreamParser) ProcessChunk(chunk []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Add chunk to buffer
	chunkText := string(chunk)
	p.buffer.WriteString(chunkText)

	// Check if we have complete lines to process
	bufferText := p.buffer.String()
	lines := strings.Split(bufferText, "\n")

	// Process all complete lines except possibly the last one
	// (which might be incomplete)
	completeLines := lines[:len(lines)-1]

	// Process each complete line
	for _, line := range completeLines {
		if err := p.processLine(line); err != nil {
			p.eventSender.SendError(err)
			return err
		}
	}

	// Keep the potentially incomplete last line in the buffer
	p.buffer.Reset()
	if len(lines) > 0 {
		p.buffer.WriteString(lines[len(lines)-1])
	}

	return nil
}

// processLine processes a single line of streaming output
func (p *SubtaskV5StreamParser) processLine(line string) error {
	trimmedLine := strings.TrimSpace(line)
	if trimmedLine == "" {
		return nil
	}

	// Regular expressions for parsing
	finalAnswerStartPattern := regexp.MustCompile(`(?i)^\s*<final_answer>\s*$`)
	finalAnswerEndPattern := regexp.MustCompile(`(?i)^\s*</final_answer>\s*$`)
	todoPattern := regexp.MustCompile(`(?i)^\s*Todo:\s*(.+)$`)
	subTodosPattern := regexp.MustCompile(`(?i)^\s*Sub-Todos:\s*$`)
	taskTitlePattern := regexp.MustCompile(`^\s*(\d+)\.\s+(.+)$`)
	objectivePattern := regexp.MustCompile(`(?i)^\s*-\s*Objective:\s*(.+)$`)
	deliverablePattern := regexp.MustCompile(`(?i)^\s*-\s*Deliverable:\s*(.+)$`)

	switch p.currentState {
	case V5State.Normal:
		if finalAnswerStartPattern.MatchString(trimmedLine) {
			p.currentState = V5State.InFinalAnswer
		}

	case V5State.InFinalAnswer:
		if finalAnswerEndPattern.MatchString(trimmedLine) {
			p.currentState = V5State.Normal
			// Send complete event
			p.eventSender.SendComplete("Subtask generation completed")
		} else if matches := todoPattern.FindStringSubmatch(trimmedLine); len(matches) > 1 {
			p.mainTodo = matches[1]
			p.eventSender.Send(models.EventRedefineTitle, p.mainTodo)
			p.currentState = V5State.InTodo
		}

	case V5State.InTodo:
		if subTodosPattern.MatchString(trimmedLine) {
			p.currentState = V5State.InSubTodos
		}

	case V5State.InSubTodos:
		if matches := taskTitlePattern.FindStringSubmatch(trimmedLine); len(matches) > 2 {
			// If we were in the middle of a task, end it
			if p.currentTaskNumber > 0 {
				p.finalizeCurrentTask()
			}

			// Start new task
			num, err := strconv.Atoi(matches[1])
			if err != nil {
				return fmt.Errorf("invalid task number: %w", err)
			}
			p.currentTaskNumber = num
			p.currentTaskTitle = matches[2]
			p.currentObjective.Reset()
			p.currentDeliverable.Reset()

			// Send task start event
			p.eventSender.Send(models.EventTaskStart,
				fmt.Sprintf("%d|%s", p.currentTaskNumber, p.currentTaskTitle))

			p.currentState = V5State.InTask
		}

	case V5State.InTask:
		if matches := objectivePattern.FindStringSubmatch(trimmedLine); len(matches) > 1 {
			p.currentObjective.WriteString(matches[1])
			p.currentState = V5State.InObjective
		} else if matches := taskTitlePattern.FindStringSubmatch(trimmedLine); len(matches) > 2 {
			// We've encountered a new task without completing the current one
			// Finalize current task and start a new one
			p.finalizeCurrentTask()

			num, err := strconv.Atoi(matches[1])
			if err != nil {
				return fmt.Errorf("invalid task number: %w", err)
			}
			p.currentTaskNumber = num
			p.currentTaskTitle = matches[2]
			p.currentObjective.Reset()
			p.currentDeliverable.Reset()

			// Send task start event
			p.eventSender.Send(models.EventTaskStart,
				fmt.Sprintf("%d|%s", p.currentTaskNumber, p.currentTaskTitle))

			p.currentState = V5State.InTask
		}

	case V5State.InObjective:
		if matches := deliverablePattern.FindStringSubmatch(trimmedLine); len(matches) > 1 {
			// Send objective immediately
			objective := p.currentObjective.String()
			if objective != "" {
				p.eventSender.SendEventWithAdditional(models.EventTaskGoal, objective, map[string]interface{}{
					"index": p.currentTaskNumber,
				})
				p.sentObjective = true
			}

			p.currentDeliverable.WriteString(matches[1])
			p.currentState = V5State.InDeliverable
		}

	case V5State.InDeliverable:
		if matches := taskTitlePattern.FindStringSubmatch(trimmedLine); len(matches) > 2 {
			// Send deliverable immediately
			deliverable := p.currentDeliverable.String()
			if deliverable != "" {
				p.eventSender.SendEventWithAdditional(models.EventTaskDeliverable, deliverable, map[string]interface{}{
					"index": p.currentTaskNumber,
				})
				p.sentDeliverable = true
			}

			// Finalize current task and start a new one
			p.finalizeCurrentTask()

			num, err := strconv.Atoi(matches[1])
			if err != nil {
				return fmt.Errorf("invalid task number: %w", err)
			}
			p.currentTaskNumber = num
			p.currentTaskTitle = matches[2]
			p.currentObjective.Reset()
			p.currentDeliverable.Reset()
			p.sentObjective = false
			p.sentDeliverable = false

			// Send task start event
			p.eventSender.Send(models.EventTaskStart,
				fmt.Sprintf("%d|%s", p.currentTaskNumber, p.currentTaskTitle))

			p.currentState = V5State.InTask
		} else if finalAnswerEndPattern.MatchString(trimmedLine) {
			// Send deliverable immediately
			deliverable := p.currentDeliverable.String()
			if deliverable != "" {
				p.eventSender.SendEventWithAdditional(models.EventTaskDeliverable, deliverable, map[string]interface{}{
					"index": p.currentTaskNumber,
				})
				p.sentDeliverable = true
			}

			// Finalize the current task and end
			p.finalizeCurrentTask()
			p.currentState = V5State.Normal
			// Send complete event
			p.eventSender.SendComplete("Subtask generation completed")
		}
	}

	return nil
}

// finalizeCurrentTask sends goal events and task end event for the current task
func (p *SubtaskV5StreamParser) finalizeCurrentTask() {
	if p.currentTaskNumber > 0 {
		// Send objective as a goal if not already sent
		if !p.sentObjective {
			objective := p.currentObjective.String()
			if objective != "" {
				p.eventSender.SendEventWithAdditional(models.EventTaskGoal, objective, map[string]interface{}{
					"index": p.currentTaskNumber,
				})
			}
		}

		// Send deliverable as a goal if not already sent
		if !p.sentDeliverable {
			deliverable := p.currentDeliverable.String()
			if deliverable != "" {
				p.eventSender.SendEventWithAdditional(models.EventTaskDeliverable, deliverable, map[string]interface{}{
					"index": p.currentTaskNumber,
				})
			}
		}

		// Reset the flags
		p.sentObjective = false
		p.sentDeliverable = false

		// Send task end event
		p.eventSender.Send(models.EventTaskEnd, fmt.Sprintf("%d", p.currentTaskNumber))
	}
}

// Finalize processes any remaining content in the buffer and finalizes the state
func (p *SubtaskV5StreamParser) Finalize() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Process any remaining content in the buffer
	if p.buffer.Len() > 0 {
		remainingText := p.buffer.String()
		lines := strings.Split(remainingText, "\n")
		for _, line := range lines {
			if line != "" {
				if err := p.processLine(line); err != nil {
					p.eventSender.SendError(err)
					return err
				}
			}
		}
	}

	// Finalize the final task if we were in one
	if (p.currentState == V5State.InTask ||
		p.currentState == V5State.InObjective ||
		p.currentState == V5State.InDeliverable) &&
		p.currentTaskNumber > 0 {
		p.finalizeCurrentTask()
	}

	// If we ended abruptly, send a completion event
	if p.currentState != V5State.Normal {
		p.eventSender.SendComplete("Subtask generation completed")
	}

	return nil
}
