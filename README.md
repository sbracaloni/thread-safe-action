Thread Safe Action Handler
==========================

Why deal with locks when you can have a lock free implementation.

The locks can be useful sometimes and not that bad for a simple problem but they can become a real nightmare 
for more complex implementations.


This library provides a simple handler to be able to execute some functions in a thread-safe context
using the Go channels.

Usage summary
-------------

Organise your code in view to extract the code to run in a thread safe context.
The thread safe functions must have the following signature:
```go
type ThreadSafeTask func(interface{}) (interface{}, error)
```

Execute those functions and wait for the result using:

```go
// SynchronousActionSend a task to be executed in a thread-safe context
SynchronousActionSend(threadSafeTask ThreadSafeTask, args interface{}) (interface{}, error)
```

Or execute those functions asynchronously:

```go
// AsynchronousActionSend a task to be executed in a thread-safe context
AsynchronousActionSend(ctrlThreadSafeFunc ThreadSafeTask, args interface{})
```

Example:
```go
func (s *MyStruc) updateAMap(args interface{}) (interface{}, error) {
	newArgs := args.(myArgsStruct)
	subByID, exists := s.theMap[newArgs.theKey]
	if !exists {
		s.subsByTheme[newArgs.theKey] = newArgs.theValue

	    return newArgs.theValue, nil
	}
	return nil, fmt.Errorf("Key %s already exists", newArgs.theKey)
}

func (s *MyStruc) AddNewSomething(key, value string) (string, error) {
	// Update the map in a thread safe environment
	reply, err := s.threadSafeActionHandler.SynchronousActionSend(s.updateAMap, myArgsStruct{
        theKey: "myKey1",
        theValue: "my value 1",
    })
	if err != nil {
		return "", err
	}
	theValue := reply.(string)
	// do something with no thread safe constraint
	return theValue, nil
}
```

See also the [Examples](./examples) section


TODO
-----

- See about [generic proposal](https://go.googlesource.com/proposal/+/master/design/go2draft-contracts.md) to avoid the dynamic casts
- Add a user context to cancel a `SynchronousActionSend`
- Add configurable timeout to the thread-safe task execution