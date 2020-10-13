package action

import (
	"context"
)

// ThreadSafeActionHandlerIft interface exposing the 2 main methods
type ThreadSafeActionHandlerIft interface {
	// SynchronousActionSend a task to be executed in a thread-safe context
	SynchronousActionSend(threadSafeTask ThreadSafeTask, args interface{}) (interface{}, error)
	// AsynchronousActionSend a task to be executed in a thread-safe context
	AsynchronousActionSend(ctrlThreadSafeFunc ThreadSafeTask, args interface{})
}

// ThreadSafeTask is executed in a thread safe context
type ThreadSafeTask func(interface{}) (interface{}, error)

type controlThreadSafeContext struct {
	controlFunc ThreadSafeTask
	args        interface{}
}

func (c controlThreadSafeContext) execute() (interface{}, error) {
	return c.controlFunc(c.args)
}

type ctrlAction struct {
	ctrlThreadSafeCtx  controlThreadSafeContext
	sync               bool
	ctrlErrorChannel   chan error
	ctrlChannelReplies chan interface{}
}

// ThreadSafeActionHandler handles tasks to execute in a thread safe context
type ThreadSafeActionHandler struct {
	ctx         context.Context
	ctrlChannel chan *ctrlAction
}

// NewThreadSafeActionHandler creates a new ThreadSafeActionHandler and start the handler loop
func NewThreadSafeActionHandler(ctx context.Context) *ThreadSafeActionHandler {
	handler := &ThreadSafeActionHandler{
		ctx:         ctx,
		ctrlChannel: make(chan *ctrlAction),
	}
	go handler.handlerLoop()
	return handler
}

func (h *ThreadSafeActionHandler) handlerLoop() {
	for {
		select {
		case <-h.ctx.Done():
			return
		case ctrl := <-h.ctrlChannel:
			result, err := ctrl.ctrlThreadSafeCtx.execute()
			if ctrl.sync {
				h.handleSyncReply(ctrl, err, result)
			}
		}
	}
}

func (h *ThreadSafeActionHandler) handleSyncReply(ctrl *ctrlAction, err error, result interface{}) {
	if err != nil {
		if ctrl.ctrlErrorChannel != nil {
			defer close(ctrl.ctrlErrorChannel)
			ctrl.ctrlErrorChannel <- err
		}
	} else if ctrl.ctrlChannelReplies != nil {
		defer close(ctrl.ctrlChannelReplies)
		ctrl.ctrlChannelReplies <- result
	}
}

// SynchronousActionSend sends an action to the thread-safe action handler in a synchronous way.
// Returns the thread safe task result
func (h *ThreadSafeActionHandler) SynchronousActionSend(threadSafeTask ThreadSafeTask, args interface{}) (interface{}, error) {
	chanDone := make(chan bool)
	send := make(chan bool, 1)
	replyChan := make(chan interface{}, 1)
	errChan := make(chan error, 1)
	defer func() {
		close(chanDone)
		close(send)
	}()
	ctrlAction := &ctrlAction{
		sync: true,
		ctrlThreadSafeCtx: controlThreadSafeContext{
			controlFunc: threadSafeTask,
			args:        args,
		},
		ctrlErrorChannel:   errChan,
		ctrlChannelReplies: replyChan,
	}
	var err error
	var reply interface{}
	received := false
	go func() {
		for !received && err == nil {
			select {
			case <-h.ctx.Done():
				err = h.ctx.Err()
			case reply = <-replyChan:
				received = true
			case err = <-errChan:
			case <-send:
				h.ctrlChannel <- ctrlAction
			}
		}
		chanDone <- true
	}()
	send <- true
	<-chanDone
	return reply, err
}

// AsynchronousActionSend sends an action to the thread-safe action handler in an asynchronous way.
func (h *ThreadSafeActionHandler) AsynchronousActionSend(ctrlThreadSafeFunc ThreadSafeTask, args interface{}) {
	h.ctrlChannel <- &ctrlAction{
		sync: false,
		ctrlThreadSafeCtx: controlThreadSafeContext{
			controlFunc: ctrlThreadSafeFunc,
			args:        args,
		},
	}
}
