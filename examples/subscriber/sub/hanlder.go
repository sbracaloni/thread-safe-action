package sub

import (
	"context"
	"fmt"

	"github.com/lithammer/shortuuid/v3"
	"github.com/sbracaloni/thread-safe-action/action"
)

// SubscriptionHandler interface
type SubscriptionHandler interface {
	AddNewSubscription(theme ActivityTheme, name PersonName) (SubscriptionID, error)
	CountSubscriptionByTheme(theme ActivityTheme) (int, error)
	RemoveSubscriptionSync(theme ActivityTheme, subID SubscriptionID) error
	RemoveSubscriptionAsync(theme ActivityTheme, subID SubscriptionID)
}

// ActivityTheme represents a specific them
type ActivityTheme string

// PersonName represents a person name
type PersonName string

// SubscriptionID identifies a subscription
type SubscriptionID string

// SubscriptionHandlerLockFree allows a user to subscribe/unsubscribe to different theme in a lock free context
type SubscriptionHandlerLockFree struct {
	subsByTheme             map[ActivityTheme]map[SubscriptionID]PersonName
	ctx                     context.Context
	threadSafeActionHandler *action.ThreadSafeActionHandler
}

// NewSubscriptionHandlerLockFree initializes a new SubscriptionHandlerLockFree
func NewSubscriptionHandlerLockFree(ctx context.Context, handler *action.ThreadSafeActionHandler) *SubscriptionHandlerLockFree {
	return &SubscriptionHandlerLockFree{
		subsByTheme:             map[ActivityTheme]map[SubscriptionID]PersonName{},
		ctx:                     ctx,
		threadSafeActionHandler: handler,
	}
}

type newSubscriptionArgs struct {
	theme ActivityTheme
	name  PersonName
}

func (s *SubscriptionHandlerLockFree) addNewSubscriptionThreadSafe(args interface{}) (interface{}, error) {
	newSubArgs := args.(newSubscriptionArgs)
	subByID, exists := s.subsByTheme[newSubArgs.theme]
	if ! exists {
		subByID = map[SubscriptionID]PersonName{}
		s.subsByTheme[newSubArgs.theme] = subByID
	}
	subID := SubscriptionID(shortuuid.New())
	subByID[subID] = newSubArgs.name
	return subID, nil
}

// AddNewSubscription creates a new subscription to a theme for the given user name
func (s *SubscriptionHandlerLockFree) AddNewSubscription(theme ActivityTheme, name PersonName) (SubscriptionID, error) {
	// Update the map in a thread safe environment
	reply, err := s.threadSafeActionHandler.SynchronousActionSend(s.addNewSubscriptionThreadSafe, newSubscriptionArgs{
		theme: theme,
		name:  name,
	})
	if err != nil {
		return "", err
	}
	subID := reply.(SubscriptionID)
	// do something with no thread safe constraint
	fmt.Printf("[Not thread safe action]:: SUB ID %s: %s => %s\n", subID, name, theme)
	return subID, nil
}

type countSubscriptionArgs struct {
	theme ActivityTheme
}

func (s *SubscriptionHandlerLockFree) countSubscriptionByThemeThreadSafe(args interface{}) (interface{}, error) {
	countSubArgs := args.(countSubscriptionArgs)
	subByID, exists := s.subsByTheme[countSubArgs.theme]
	if ! exists {
		return 0, nil
	}
	return len(subByID), nil
}

// CountSubscriptionByTheme returns the number of subscriptions by theme
func (s *SubscriptionHandlerLockFree) CountSubscriptionByTheme(theme ActivityTheme) (int, error) {
	// Update the map in a thread safe environment
	reply, err := s.threadSafeActionHandler.SynchronousActionSend(s.countSubscriptionByThemeThreadSafe, countSubscriptionArgs{
		theme: theme,
	})
	if err != nil {
		return -1, err
	}
	subCount := reply.(int)
	// do something with no thread safe constraint
	return subCount, nil
}

type removeSubscriptionArgs struct {
	subID SubscriptionID
	theme ActivityTheme
}

func (s *SubscriptionHandlerLockFree) removeSubscriptionThreadSafe(args interface{}) (interface{}, error) {
	removeSubArgs := args.(removeSubscriptionArgs)
	subByID, exists := s.subsByTheme[removeSubArgs.theme]
	if exists {
		delete(subByID, removeSubArgs.subID)
		if len(subByID) == 0 {
			delete(s.subsByTheme, removeSubArgs.theme)
		}
	}
	return nil, nil
}

// RemoveSubscriptionSync deletes a the subscription associated to the subID for the given theme
func (s *SubscriptionHandlerLockFree) RemoveSubscriptionSync(theme ActivityTheme, subID SubscriptionID) error {
	// Update the map in a thread safe environment
	_, err := s.threadSafeActionHandler.SynchronousActionSend(s.removeSubscriptionThreadSafe, removeSubscriptionArgs{
		theme: theme,
		subID: subID,
	})
	if err != nil {
		return err
	}
	// do something with no thread safe constraint
	fmt.Printf("[Not thread safe action]:: removed sub %s-%s\n", subID, theme)
	return nil
}

// RemoveSubscriptionAsync sends a delete order to remove a the subscription associated to the subID for the given theme
func (s *SubscriptionHandlerLockFree) RemoveSubscriptionAsync(theme ActivityTheme, subID SubscriptionID) {
	// Update the map in a thread safe environment
	s.threadSafeActionHandler.AsynchronousActionSend(s.removeSubscriptionThreadSafe, removeSubscriptionArgs{
		theme: theme,
		subID: subID,
	})

	// do something with no thread safe constraint
	fmt.Printf("[Not thread safe action]:: asked for sub %s-%s remove\n", subID, theme)
}
