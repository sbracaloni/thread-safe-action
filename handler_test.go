package action_test

import (
	"context"
	"fmt"
	"testing"

	"gotest.tools/assert"

	action "github.com/sbracaloni/thread-safe-action"
)

// Consider using Ginko especially to be able to write parametrized tests

func Test_ShouldExecuteActionFromAnAsynchronousSend(t *testing.T) {
	handlerCtx, cancelHandler := context.WithCancel(context.TODO())
	defer cancelHandler()
	var actionHandler action.ThreadSafeActionHandlerIft = action.NewThreadSafeActionHandler(handlerCtx)

	passedArgsChan := make(chan interface{})

	threadSafeFunc := func(args interface{}) (interface{}, error) {
		passedArgsChan <- args
		return nil, nil
	}

	args := 1234
	actionHandler.AsynchronousActionSend(threadSafeFunc, args)
	passedArgs := <-passedArgsChan
	obtainedArgs := passedArgs.(int)
	assert.Equal(t, obtainedArgs, args)
}

func Test_ShouldExecuteActionFromAnAsynchronousSendWithNilArgs(t *testing.T) {
	handlerCtx, cancelHandler := context.WithCancel(context.TODO())
	defer cancelHandler()
	var actionHandler action.ThreadSafeActionHandlerIft = action.NewThreadSafeActionHandler(handlerCtx)
	hasBeenCalled := make(chan bool)
	threadSafeFunc := func(args interface{}) (interface{}, error) {
		hasBeenCalled <- true
		return nil, nil
	}

	actionHandler.AsynchronousActionSend(threadSafeFunc, nil)
	<-hasBeenCalled
}

func Test_ShouldExecuteActionFromASynchronousSend(t *testing.T) {
	handlerCtx, cancelHandler := context.WithCancel(context.TODO())
	defer cancelHandler()

	var actionHandler action.ThreadSafeActionHandlerIft = action.NewThreadSafeActionHandler(handlerCtx)

	type RetValue struct {
		providedArgs interface{}
	}

	threadSafeFunc := func(args interface{}) (interface{}, error) {
		return RetValue{providedArgs: args}, nil
	}

	args := 1234
	result, err := actionHandler.SynchronousActionSend(threadSafeFunc, args)
	assert.NilError(t, err)
	assert.Equal(t, result, RetValue{providedArgs: args})
}

func Test_ShouldExecuteActionFromASynchronousSendWithNilArgs(t *testing.T) {
	handlerCtx, cancelHandler := context.WithCancel(context.TODO())
	defer cancelHandler()
	var actionHandler action.ThreadSafeActionHandlerIft = action.NewThreadSafeActionHandler(handlerCtx)
	type RetValue struct {
		providedArgs interface{}
	}

	threadSafeFunc := func(args interface{}) (interface{}, error) {
		return RetValue{providedArgs: args}, nil
	}

	result, err := actionHandler.SynchronousActionSend(threadSafeFunc, nil)
	assert.NilError(t, err)
	assert.Equal(t, result, RetValue{})
}

func Test_ShouldExecuteActionFromASynchronousSendAnReturnError(t *testing.T) {
	handlerCtx, cancelHandler := context.WithCancel(context.TODO())
	defer cancelHandler()

	var actionHandler action.ThreadSafeActionHandlerIft = action.NewThreadSafeActionHandler(handlerCtx)

	errMsg := "something wrong happened"
	threadSafeFunc := func(args interface{}) (interface{}, error) {
		return nil, fmt.Errorf(errMsg)
	}
	result, err := actionHandler.SynchronousActionSend(threadSafeFunc, nil)
	assert.Equal(t, result, nil)
	assert.Error(t, err, errMsg)
}

func Test_ShouldStopTaskExecutionWhenHandlerContextIsCancelledDuringSynchronousSend(t *testing.T) {
	handlerCtx, cancelHandler := context.WithCancel(context.TODO())
	var actionHandler action.ThreadSafeActionHandlerIft = action.NewThreadSafeActionHandler(handlerCtx)
	done := make(chan bool)
	threadSafeFunc := func(args interface{}) (interface{}, error) {
		cancelHandler()
		<-done
		return nil, nil
	}

	result, err := actionHandler.SynchronousActionSend(threadSafeFunc, nil)

	assert.Error(t, err, "context canceled")

	assert.Equal(t, result, nil)
	done <- true
}

func Test_ShouldStopTaskExecutionWhenHandlerContextIsCancelledDuringAsynchronousSend(t *testing.T) {
	handlerCtx, cancelHandler := context.WithCancel(context.TODO())
	var actionHandler action.ThreadSafeActionHandlerIft = action.NewThreadSafeActionHandler(handlerCtx)
	done := make(chan bool)

	hasBeenCalled := make(chan bool)
	threadSafeFunc := func(args interface{}) (interface{}, error) {
		hasBeenCalled <- true
		<-done
		return nil, nil
	}

	actionHandler.AsynchronousActionSend(threadSafeFunc, nil)
	<-hasBeenCalled
	cancelHandler()
	done <- true
}

func Test_ShouldReturnErrorForAllTheWaitingSynchronousTaskIfTheContextIsCanceled(t *testing.T) {
	handlerCtx, cancelHandler := context.WithCancel(context.TODO())
	handlerCtx.Done()
	var actionHandler action.ThreadSafeActionHandlerIft = action.NewThreadSafeActionHandler(handlerCtx)
	done := make(chan bool)

	nbConc := 100
	allCanceled := make(chan bool, nbConc)
	hasBeenCalled := make(chan bool)
	threadSafeFunc := func(args interface{}) (interface{}, error) {
		hasBeenCalled <- true
		<-done
		return nil, nil
	}
	doNothingTask := func(args interface{}) (interface{}, error) {
		return nil, nil
	}

	actionHandler.AsynchronousActionSend(threadSafeFunc, nil)
	<-hasBeenCalled
	for i := 0; i < nbConc; i++ {
		go func() {
			// Those actions will wait to send on the control channel. They should be canceled if the context is done
			// (even if they are already waiting in the queue)
			_, err := actionHandler.SynchronousActionSend(doNothingTask, nil)
			assert.Error(t, err, "context canceled")
			allCanceled <- true
		}()
	}
	cancelHandler()
	done <- true
	for i := 0; i < nbConc; i++ {
		<-allCanceled
	}
}

func Test_ShouldDiscardSendForAllTheWaitingAsynchronousTaskIfTheContextIsCanceled(t *testing.T) {
	handlerCtx, cancelHandler := context.WithCancel(context.TODO())
	handlerCtx.Done()
	var actionHandler action.ThreadSafeActionHandlerIft = action.NewThreadSafeActionHandler(handlerCtx)
	done := make(chan bool)

	hasBeenCalled := make(chan bool)
	threadSafeFunc := func(args interface{}) (interface{}, error) {
		hasBeenCalled <- true
		<-done
		return nil, nil
	}
	panicTask := func(args interface{}) (interface{}, error) {
		panic("Should not be triggered")
		return nil, nil
	}
	actionHandler.AsynchronousActionSend(threadSafeFunc, nil)
	<-hasBeenCalled
	cancelHandler()
	// This task should not be sent an return immediately
	actionHandler.AsynchronousActionSend(panicTask, nil)
	done <- true
}
