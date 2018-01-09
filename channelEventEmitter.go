package emitter

import (
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"sync"
)

var eventHandlerCounter int

type valuesResult []reflect.Value

// Emitter of events
type Emitter struct {
	events map[string]*eventHandler
}

// Receiver of event handlers
type Receiver struct {
	FuncName string
	Values   []interface{}
}

type handler struct {
	name string
	fn   reflect.Value
}

type eventHandler struct {
	token       int
	handlers    []handler
	emitChan    chan []interface{}
	receiveChan chan Receiver
	doneChan    chan bool
}

func (eH *eventHandler) listen() {
	go func() {
		var w sync.WaitGroup

		for a := range eH.emitChan {
			w.Add(1)
			args := make([]reflect.Value, len(a))

			for i, arg := range a {
				args[i] = reflect.ValueOf(arg)
			}

			go func() {
				var wG sync.WaitGroup

				for _, h := range eH.handlers {
					wG.Add(1)

					if isValidFuncForEvent(h.fn, args) {
						wG.Add(1)

						go func() {
							results := h.fn.Call(args)
							eH.receiveChan <- valuesResult(results).receiver(h)
							wG.Done()
						}()
					}

					wG.Done()
				}

				wG.Wait()
				w.Done()
			}()
		}

		w.Wait()
		eH.doneChan <- true
	}()
}

func isValidFuncForEvent(fn reflect.Value, args []reflect.Value) (isValid bool) {
	isValid = false
	isVariadic := fn.Type().IsVariadic()
	argsLen := len(args)
	fnNumIn := fn.Type().NumIn()

	if (!isVariadic && argsLen == fnNumIn) || (isVariadic && argsLen >= fnNumIn) {
		isValid = true

		for i := 0; i < fnNumIn; i++ {
			argType := args[i].Type()
			fnInType := fn.Type().In(i)

			if isVariadic && (i+1) == fnNumIn {
				argType = reflect.SliceOf(argType)
			}

			isValid = argType == fnInType

			if !isValid {
				break
			}
		}
	}

	return
}

func (vArr valuesResult) receiver(h handler) (r Receiver) {
	r.FuncName = h.name

	for _, value := range vArr {
		r.Values = append(r.Values, value.Interface())
	}

	return
}

func (e Emitter) eventExists(event string) (eH *eventHandler, exists bool, err error) {
	if eH, exists = e.events[event]; !exists {
		err = fmt.Errorf("emitter: %s are not registered", event)
	}

	return
}

func nameOfFunc(v reflect.Value) string {
	if v.Kind() == reflect.Func {
		if rf := runtime.FuncForPC(v.Pointer()); rf != nil {
			return rf.Name()
		}
	}

	return v.String()
}

func (r Receiver) String() string {
	return fmt.Sprintf("Func: %s, Returned values: %v", r.FuncName, r.Values)
}

// New Emitter pointer
func New() (e *Emitter) {
	e = &Emitter{events: make(map[string]*eventHandler)}

	return
}

// Register an event and it's receivers, or only the receivers if event is already registered
// and returns it's channels to emit and receive
func (e *Emitter) Register(event string, fns ...interface{}) (emit chan<- []interface{}, receive <-chan Receiver, err error) {
	var (
		eH     *eventHandler
		exists bool
	)

	if eH, exists, _ = e.eventExists(event); !exists {
		eventHandlerCounter++

		eH = &eventHandler{
			token:       eventHandlerCounter,
			handlers:    make([]handler, 0),
			emitChan:    make(chan []interface{}),
			receiveChan: make(chan Receiver),
			doneChan:    make(chan bool),
		}

		e.events[event] = eH

		defer eH.listen()
	}

	errChan := make(chan string)

	go func() {
		for i, fn := range fns {
			if reflect.TypeOf(fn).Kind() != reflect.Func {
				errChan <- fmt.Sprintf("Argument %d is not of type reflect.Func and was not registered", i)

				continue
			}

			v := reflect.ValueOf(fn)
			eH.handlers = append(eH.handlers, handler{name: nameOfFunc(v), fn: v})
		}

		close(errChan)
	}()

	var errMsg string

	for err := range errChan {
		errMsg = fmt.Sprintln(errMsg, err)
	}

	emit = eH.emitChan
	receive = eH.receiveChan
	err = fmt.Errorf("emitter: %s", errMsg)

	return
}

// UnregisterEvent an event and close it's channels
func (e *Emitter) UnregisterEvent(event string) (err error) {
	var (
		eH     *eventHandler
		exists bool
	)

	if eH, exists, err = e.eventExists(event); exists {
		go func() {
			defer delete(e.events, event)
			eH.handlers = nil
			close(eH.emitChan)
			<-eH.doneChan
			close(eH.doneChan)
			close(eH.receiveChan)
			eH = nil
		}()
	}

	return
}

// Unregister the receivers for the specified event
func (e *Emitter) Unregister(event string, fns ...interface{}) (err error) {
	var (
		eH     *eventHandler
		exists bool
	)

	if eH, exists, err = e.eventExists(event); exists {
		for _, fn := range fns {
			iSlice := make([]int, 0)

			for i, h := range eH.handlers {
				if h.fn == reflect.ValueOf(fn) {
					iSlice = append(iSlice, i)
				}
			}

			if len(iSlice) > 0 {
				for _, iToRemove := range iSlice {
					copy(eH.handlers[iToRemove:], eH.handlers[iToRemove+1:])
				}

				eH.handlers = eH.handlers[:len(eH.handlers)-len(iSlice)]
			}
		}
	}

	return
}

// Channels to emit and receive from specified event
func (e *Emitter) Channels(event string) (emit chan<- []interface{}, receive <-chan Receiver, err error) {
	var (
		eH     *eventHandler
		exists bool
	)

	if eH, exists, err = e.eventExists(event); exists {
		emit = eH.emitChan
		receive = eH.receiveChan
	}

	return
}

// Events registereds
func (e *Emitter) Events() (events []string, err error) {
	if len(e.events) != 0 {
		keys := reflect.ValueOf(e.events).MapKeys()
		events = make([]string, len(keys))

		for i, k := range keys {
			events[i] = k.String()
		}
	} else {
		err = errors.New("emitter: There are not events registereds")
	}

	return
}
