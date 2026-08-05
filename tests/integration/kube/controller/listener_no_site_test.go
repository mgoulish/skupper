//go:build integration

package kubecontrollertest

import (
	"context"
	"testing"
	"time"

	"github.com/skupperproject/skupper/internal/fixtures"
	"gotest.tools/v3/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	skupperv2alpha1 "github.com/skupperproject/skupper/pkg/apis/skupper/v2alpha1"
)

// This test makes sure that if a user tries to define a Listener
// when there is no Site already defined, Skupper should throw an error.

func TestListenerWithoutSite(t *testing.T) {
	tc := setup(t)
	namespace := "listener-no-site"
	tc.createNamespace(namespace)

	ctx := context.Background()

	// Create a Listener in our namespace
	// that we have not created a Site in.
	listener := listenerWithHostPort("test-listener", namespace, "test-service", 8080)
	_, err := tc.clients.GetSkupperClient().SkupperV2alpha1().Listeners(namespace).Create(ctx, listener, metav1.CreateOptions{})
	assert.NilError(t, err)

	// Try every quarter-second until we get a Status Error,
	// or run out of patience and fail.
	// And make sure that the Status Error (written by the
	// Controller) is what we expect.
	var actual *skupperv2alpha1.Listener
	waitFor(t, 30*time.Second, 250*time.Millisecond, func() (bool, error) {
		var err error
		actual, err = tc.clients.GetSkupperClient().SkupperV2alpha1().Listeners(namespace).Get(ctx, "test-listener", metav1.GetOptions{})
		if done, err := retryOnNotFound(err); !done {
			return false, err // Keep trying, unless real error, then fail the test right here.
			// (Because of an assert in waitFor.)
		}

		if actual.Status.StatusType == skupperv2alpha1.StatusError {
			return true, nil // We have the Status Error -- quit retrying
		}
		return false, nil // No Status Error yet -- keep retrying
	})

	// If we got the expected error message, succeed. Else fail.
	verifyStatus(t,
		fixtures.Status(skupperv2alpha1.StatusError, "No active site in namespace"),
		actual.Status.Status,
	)
}
