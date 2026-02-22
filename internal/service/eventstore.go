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
}

func newDeploymentEventStore() *deploymentEventStore {
	return &deploymentEventStore{
		events:      make(map[uuid.UUID][]domain.DeploymentEvent),
		subscribers: make(map[uuid.UUID][]chan domain.DeploymentEvent),
		completed:   make(map[uuid.UUID]bool),
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
func (s *deploymentEventStore) Subscribe(deploymentID uuid.UUID) ([]domain.DeploymentEvent, chan domain.DeploymentEvent, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Copy stored events
	stored := make([]domain.DeploymentEvent, len(s.events[deploymentID]))
	copy(stored, s.events[deploymentID])

	if s.completed[deploymentID] {
		return stored, nil, true
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
