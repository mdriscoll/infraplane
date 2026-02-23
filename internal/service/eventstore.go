package service

import (
	"sync"

	"github.com/google/uuid"
	"github.com/matthewdriscoll/infraplane/internal/domain"
)

// deploymentEventStore stores deployment events in memory and supports
// live subscriptions so clients can reconnect to in-progress deployments.
type deploymentEventStore struct {
	mu          sync.RWMutex
	events      map[uuid.UUID][]domain.DeploymentEvent
	subscribers map[uuid.UUID][]chan domain.DeploymentEvent
	completed   map[uuid.UUID]bool
	paused      map[uuid.UUID]bool
}

func newDeploymentEventStore() *deploymentEventStore {
	return &deploymentEventStore{
		events:      make(map[uuid.UUID][]domain.DeploymentEvent),
		subscribers: make(map[uuid.UUID][]chan domain.DeploymentEvent),
		completed:   make(map[uuid.UUID]bool),
		paused:      make(map[uuid.UUID]bool),
	}
}

// Append stores an event and broadcasts it to all subscribers.
func (s *deploymentEventStore) Append(deploymentID uuid.UUID, event domain.DeploymentEvent) {
	s.mu.Lock()
	s.events[deploymentID] = append(s.events[deploymentID], event)
	subs := s.subscribers[deploymentID]
	s.mu.Unlock()

	// Non-blocking broadcast to subscribers
	for _, ch := range subs {
		select {
		case ch <- event:
		default:
			// subscriber is slow — skip to avoid blocking the deploy pipeline
		}
	}
}

// MarkComplete marks a deployment's event stream as finished and closes all subscriber channels.
func (s *deploymentEventStore) MarkComplete(deploymentID uuid.UUID) {
	s.mu.Lock()
	s.completed[deploymentID] = true
	subs := s.subscribers[deploymentID]
	delete(s.subscribers, deploymentID)
	s.mu.Unlock()

	for _, ch := range subs {
		close(ch)
	}
}

// IsComplete returns true if the deployment's event stream has finished.
func (s *deploymentEventStore) IsComplete(deploymentID uuid.UUID) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.completed[deploymentID]
}

// MarkPaused marks a deployment's event stream as paused (awaiting approval).
// Closes all subscriber channels but does NOT mark as completed so reconnecting
// clients know the deployment is not finished.
func (s *deploymentEventStore) MarkPaused(deploymentID uuid.UUID) {
	s.mu.Lock()
	s.paused[deploymentID] = true
	subs := s.subscribers[deploymentID]
	delete(s.subscribers, deploymentID)
	s.mu.Unlock()

	for _, ch := range subs {
		close(ch)
	}
}

// MarkActive clears the paused flag so new subscribers can receive live events
// during the resume/apply phase.
func (s *deploymentEventStore) MarkActive(deploymentID uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.paused, deploymentID)
}

// IsPaused returns true if the deployment is paused (awaiting approval).
func (s *deploymentEventStore) IsPaused(deploymentID uuid.UUID) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.paused[deploymentID]
}

// GetEvents returns all stored events for a deployment.
func (s *deploymentEventStore) GetEvents(deploymentID uuid.UUID) []domain.DeploymentEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	events := s.events[deploymentID]
	result := make([]domain.DeploymentEvent, len(events))
	copy(result, events)
	return result
}

// Subscribe returns existing events and a channel for future events.
// The channel will be closed when the deployment completes.
// Returns (storedEvents, liveChan, alreadyComplete).
// When paused (awaiting approval): returns stored events, nil channel, false.
// This tells the caller that events exist but the deployment is not complete.
func (s *deploymentEventStore) Subscribe(deploymentID uuid.UUID) ([]domain.DeploymentEvent, chan domain.DeploymentEvent, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Copy stored events
	stored := make([]domain.DeploymentEvent, len(s.events[deploymentID]))
	copy(stored, s.events[deploymentID])

	if s.completed[deploymentID] {
		return stored, nil, true
	}

	// If paused (awaiting approval), return stored events but no live channel.
	// The caller sees: stored events, nil channel, NOT complete = paused state.
	if s.paused[deploymentID] {
		return stored, nil, false
	}

	// Create subscriber channel
	ch := make(chan domain.DeploymentEvent, 32)
	s.subscribers[deploymentID] = append(s.subscribers[deploymentID], ch)

	return stored, ch, false
}

// Unsubscribe removes a specific channel from the subscriber list.
func (s *deploymentEventStore) Unsubscribe(deploymentID uuid.UUID, ch chan domain.DeploymentEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	subs := s.subscribers[deploymentID]
	for i, sub := range subs {
		if sub == ch {
			s.subscribers[deploymentID] = append(subs[:i], subs[i+1:]...)
			break
		}
	}
}
