package emitter

import "testing"

func TestRegister(t *testing.T) {
	e := New()

	event := "EVENT"

	emitter, receiver, _ := e.Register(event)

	if emitter == nil {
		t.Error("Null emit channel")
	}

	if receiver == nil {
		t.Error("Null receive channel")
	}

	emitter, receiver, _ = e.Register(event)

	events, _ := e.Events()

	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}

	_, _, err := e.Register(event, 1, 2)

	if err == nil {
		t.Error("Expected error registering non-function")
	}
}

func TestEvents(t *testing.T) {
	e := New()

	events, err := e.Events()

	if len(events) != 0 {
		t.Errorf("Expected 0 event, got %d", len(events))
	}

	event := "EVENT"

	e.Register(event)
	events, err = e.Events()

	if err != nil {
		t.Error(err)
	}

	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}

	e.Register(event)
	events, err = e.Events()

	if err != nil {
		t.Error(err)
	}

	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}

	e.Register("NEW")
	events, err = e.Events()

	if err != nil {
		t.Error(err)
	}

	if len(events) != 2 {
		t.Errorf("Expected 2 event, got %d", len(events))
	}
}

func TestChannels(t *testing.T) {
	e := New()

	event := "EVENT"
	e.Register(event)
	emitter, receiver, err := e.Channels(event)

	if err != nil {
		t.Error(err)
	}

	if emitter == nil {
		t.Errorf("Null emit channel for event %s", event)
	}

	if receiver == nil {
		t.Errorf("Null receive channel for event %s", event)
	}

	event = "NEW"
	e.Register(event)
	emitter, receiver, err = e.Channels(event)

	if err != nil {
		t.Error(err)
	}

	if emitter == nil {
		t.Errorf("Null emit channel for event %s", event)
	}

	if receiver == nil {
		t.Errorf("Null receive  channel for event %s", event)
	}
}

func TestUnregisterChannel(t *testing.T) {
	e := New()

	event := "EVENT"
	e.Register(event)

	events, err := e.Events()

	if err != nil {
		t.Error(err)
	}

	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}

	err = e.UnregisterEvent(event)

	if err != nil {
		t.Error(err)
	}

	events, err = e.Events()

	if len(events) != 0 {
		t.Errorf("Expected 0 event, got %d", len(events))
	}
}

func TestEmitAndReceive(t *testing.T) {
	e := New()

	event := "EVENT"

	emitter, receiver, _ := e.Register(event)

	emitter <- []interface{}{1}

	if r := <-receiver; r.FuncName != "" {
		t.Errorf("Expected no function to execute, got %s", r.FuncName)
	}

	param0Out0Func := func() {

	}

	emitter, receiver, _ = e.Register(event, param0Out0Func)

	emitter <- make([]interface{}, 0)

	if r := <-receiver; len(r.Values) != 0 {
		t.Errorf("Expected no values from receiver, got %v", r.Values)
	}

	e.Unregister(event, param0Out0Func)

	param1Out0Func := func(i int) {

	}

	emitter, receiver, _ = e.Register(event, param1Out0Func)

	emitter <- []interface{}{1}

	if r := <-receiver; len(r.Values) != 0 {
		t.Errorf("Expected no values from receiver, got %v", r.Values)
	}

	e.Unregister(event, param1Out0Func)

	param1Out1Func := func(i int) int {
		return 1
	}

	emitter, receiver, _ = e.Register(event, param1Out1Func)

	emitter <- []interface{}{1}

	close(emitter)

	if r := <-receiver; len(r.Values) != 1 || r.Values[0] != 1 {
		t.Errorf("Expected [1] from receiver, got %v", r.Values)
	}
}
