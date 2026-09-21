package site

import (
	"errors"
	"testing"

	"github.com/skupperproject/skupper/internal/qdr"
	skupperv2alpha1 "github.com/skupperproject/skupper/pkg/apis/skupper/v2alpha1"
	"gotest.tools/v3/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// fakeBindingContext is a BindingContext that returns a fixed error from Expose.
// Select and Unexpose are unused in these tests.
type fakeBindingContext struct {
	exposeErr error
}

// This is only here so that fakeBindingContext will be a proper BindingContext
func (f *fakeBindingContext) Select(connector *skupperv2alpha1.Connector) TargetSelection {
	return nil
}

func (f *fakeBindingContext) Expose(ports *ExposedPortSet) error {
	return f.exposeErr
}

// This is only here so that fakeBindingContext will be a proper BindingContext
func (f *fakeBindingContext) Unexpose(host string) error {
	return nil
}

// testListener builds a tiny Listener Custom Resource for UpdateListener.
func testListener(name, host string) *skupperv2alpha1.Listener {
	return &skupperv2alpha1.Listener{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "ns"},
		Spec: skupperv2alpha1.ListenerSpec{
			Host:       host,
			Port:       8080,
			RoutingKey: name,
		},
	}
}

// newBindingsForExposeTest builds ExtendedBindings with a fake Expose
// and the event handler wired up so that ListenerUpdated will still run.
func newBindingsForExposeTest(exposeErr error) *ExtendedBindings {
	eb := NewExtendedBindings(nil, "/tmp/profiles")
	eb.context = &fakeBindingContext{exposeErr: exposeErr}
	eb.mapping = qdr.RecoverPortMapping(nil)
	if eb.mapping == nil {
		t := qdr.RecoverPortMapping(&qdr.RouterConfig{})
		eb.mapping = t
	}
	if eb.exposed == nil {
		eb.exposed = ExposedPorts{}
	}
	if eb.lastListenerExposeErr == nil {
		eb.lastListenerExposeErr = map[string]error{}
	}
	eb.bindings.SetBindingEventHandler(eb)
	return eb
}

// TestUpdateListenerPropagatesExposeErrorByName checks that a failed
// Service exposure gets returned from UpdateListener --
// and is then removed from lastListenerExposeErr.
func TestUpdateListenerPropagatesExposeErrorByName(t *testing.T) {
	exposeErr := errors.New("Service name too long")
	eb := newBindingsForExposeTest(exposeErr)

	// Update listener returns the error.
	_, err := eb.UpdateListener("listener-a", testListener("listener-a", "svc-a"))
	assert.ErrorContains(t, err, "Service name too long")

	// And removed it from the map of each Listener's most recent error.
	_, still_there := eb.lastListenerExposeErr["listener-a"]
	assert.Assert(t, !still_there, "error for listener-a should be consumed")
}

// TestUpdateListenerExposeErrorsDoNotCrossListeners checks that an
// expose error for listener-a is not returned as listener-b's error.
func TestUpdateListenerExposeErrorsDoNotCrossListeners(t *testing.T) {
	eb := newBindingsForExposeTest(errors.New("fail-a"))

	_, errA := eb.UpdateListener("listener-a", testListener("listener-a", "svc-a"))
	assert.ErrorContains(t, errA, "fail-a")

	eb.context = &fakeBindingContext{exposeErr: errors.New("fail-b")}
	_, errB := eb.UpdateListener("listener-b", testListener("listener-b", "svc-b"))
	assert.ErrorContains(t, errB, "fail-b")
}
