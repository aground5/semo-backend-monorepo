package streamparser

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"semo-server/internal/ai/models"
	"semo-server/internal/ai/streaming"
)

// ParserState represents the current state of the streaming parser
type ParserState int

const (
	StateNormal ParserState = iota
	StateInTask
	StateAfterRedefineHeader
	StateInRedefineTitle
	StateInRedefineGoal
)

// SubtaskStreamParser processes streaming subtask output
type SubtaskStreamParser struct {
	buffer            strings.Builder
	currentState      ParserState
	currentTaskNumber string
	currentTaskTitle  string
	redefineTitle     strings.Builder
	eventSender       *streaming.EventSender
	mu                sync.Mutex
}

// NewSubtaskStreamParser creates a new SubtaskStreamParser
func NewSubtaskStreamParser(streamChan chan<- string) *SubtaskStreamParser {
	return &SubtaskStreamParser{
		buffer:            strings.Builder{},
		currentState:      StateNormal,
		currentTaskNumber: "",
		currentTaskTitle:  "",
		redefineTitle:     strings.Builder{},
		eventSender:       streaming.NewEventSender(streamChan),
		mu:                sync.Mutex{},
	}
}

// GetType returns the type of parser
func (p *SubtaskStreamParser) GetType() string {
	return "subtask_stream_parser"
}

// ProcessChunk processes a chunk of streaming output
func (p *SubtaskStreamParser) ProcessChunk(chunk []byte) error {
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
func (p *SubtaskStreamParser) processLine(line string) error {
	trimmedLine := strings.TrimSpace(line)
	if trimmedLine == "" {
		return nil
	}

	// Regular expressions for parsing
	taskTitlePattern := regexp.MustCompile(`^\s*(\d+)\.\s+(.+)$`)
	taskGoalPattern := regexp.MustCompile(`^\s*-\s+(.+)$`)
	redefineHeaderPattern := regexp.MustCompile(`(?i)^redefine\s+to-do:$`)

	switch p.currentState {
	case StateNormal:
		// Check if this line starts a task
		if matches := taskTitlePattern.FindStringSubmatch(trimmedLine); len(matches) > 2 {
			// If we were in a task before, close it (shouldn't happen in StateNormal, but just in case)
			if p.currentTaskNumber != "" {
				p.eventSender.Send(models.EventTaskEnd, p.currentTaskNumber)
			}

			// Extract task number and title
			p.currentTaskNumber = matches[1]
			p.currentTaskTitle = matches[2]

			// Send task start event with number and title
			p.eventSender.Send(models.EventTaskStart,
				fmt.Sprintf("%s|%s", p.currentTaskNumber, p.currentTaskTitle))
			p.currentState = StateInTask
		} else if redefineHeaderPattern.MatchString(trimmedLine) {
			// We've found the "Redefine task:" header
			p.currentState = StateAfterRedefineHeader
		}

	case StateInTask:
		// Check if this is a task goal line (starts with dash)
		if matches := taskGoalPattern.FindStringSubmatch(trimmedLine); len(matches) > 1 {
			goalText := matches[1]
			// Send goal event
			p.eventSender.Send(models.EventTaskGoal, goalText)
		} else if matches := taskTitlePattern.FindStringSubmatch(trimmedLine); len(matches) > 2 {
			// We've found a new task title, close the current one and start a new one
			p.eventSender.Send(models.EventTaskEnd, p.currentTaskNumber)

			p.currentTaskNumber = matches[1]
			p.currentTaskTitle = matches[2]
			p.eventSender.Send(models.EventTaskStart,
				fmt.Sprintf("%s|%s", p.currentTaskNumber, p.currentTaskTitle))
		} else if redefineHeaderPattern.MatchString(trimmedLine) {
			// We've found the "Redefine task:" header, close the current task
			p.eventSender.Send(models.EventTaskEnd, p.currentTaskNumber)
			p.currentState = StateAfterRedefineHeader
		}

	case StateAfterRedefineHeader:
		// The next non-empty line after "Redefine task:" is the redefined title
		p.redefineTitle.Reset()
		p.redefineTitle.WriteString(trimmedLine)
		p.eventSender.Send(models.EventRedefineTitle, trimmedLine)
		p.currentState = StateInRedefineTitle

	case StateInRedefineTitle:
		// After the title, the next line(s) are the redefined goal
		p.eventSender.Send(models.EventRedefineGoal, trimmedLine)
		p.currentState = StateInRedefineGoal

	case StateInRedefineGoal:
		// Continue capturing lines for the redefined goal
		p.eventSender.Send(models.EventRedefineGoal, trimmedLine)
	}

	return nil
}

// Finalize processes any remaining content in the buffer and finalizes the state
func (p *SubtaskStreamParser) Finalize() error {
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

	// Close the final task if we were in one
	if p.currentState == StateInTask && p.currentTaskNumber != "" {
		p.eventSender.Send("task_end", p.currentTaskNumber)
	}

	return nil
}