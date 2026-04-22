package kubernetes

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/meshery/meshery/server/models"
	"github.com/meshery/meshkit/logger"
	"github.com/meshery/schemas/models/core"
)

// TestAssignInitialCtx_AttachesLoggerBeforeClientSetAssignment guards against
// a nil-pointer panic on login when a persisted K8s context can't be reached:
// GenerateKubeHandler errors → GenerateK8sClientSet hits its log.Warn path →
// interface-method-on-nil panic.
//
// The bug was an ordering mistake in AssignInitialCtx: machinectx.log was
// assigned AFTER AssignClientSetToContext, which threaded the still-nil
// log through GenerateClientSetAction → GenerateK8sClientSet. Any persisted
// context whose API server wasn't reachable (common: stale contexts from a
// remote provider pointing at clusters this host can't route to) produced
// the panic on every /api request that went through K8sFSMMiddleware.
//
// This test exercises AssignInitialCtx with a K8sContext whose API server
// is unreachable, which forces the error path previously responsible for
// the nil-deref panic. The assertions:
//   - no panic
//   - machinectx.log is the logger we passed (proving the attach happened
//     before any action could consume it)
func TestAssignInitialCtx_AttachesLoggerBeforeClientSetAssignment(t *testing.T) {
	log, err := logger.New("test", logger.Options{})
	if err != nil {
		t.Fatalf("failed to build test logger: %v", err)
	}

	user := &models.User{}
	if userID, err := uuid.NewV4(); err == nil {
		user.ID = core.Uuid(userID)
	}

	sysID := core.Uuid(uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"))

	ctx := context.Background()
	ctx = context.WithValue(ctx, models.UserCtxKey, user)
	ctx = context.WithValue(ctx, models.SystemIDKey, &sysID)
	// ProviderCtxKey: a typed-nil is fine — AssignControllerHandlers is only
	// reached after AssignClientSetToContext, and that's the point we want
	// to defend. If AssignClientSetToContext returns an error we never reach
	// controller setup, which matches the production scenario.
	var provider models.Provider
	ctx = context.WithValue(ctx, models.ProviderCtxKey, provider)

	machinectx := &MachineCtx{
		K8sContext: models.K8sContext{
			// Deliberately empty: any unreachable/invalid kubeconfig is fine.
			// The point is to force GenerateKubeHandler to fail so the
			// previously panicking log.Warn path runs.
			Name:         "unreachable-test-context",
			Server:       "https://127.0.0.1:1", // RFC-reserved, refused instantly
			ConnectionID: uuid.Must(uuid.NewV4()).String(),
		},
		// clientset left nil to force AssignClientSetToContext to attempt
		// GenerateClientSetAction (the panicking path).
	}

	result, _, err := AssignInitialCtx(ctx, machinectx, log)

	// The production repro errors out of AssignClientSetToContext; we accept
	// any error or nil here — what we're asserting is *the absence of a
	// panic* and that the log attachment happened before that error path ran.
	_ = err
	_ = result

	if machinectx.log == nil {
		t.Fatal("expected machinectx.log to be populated before AssignClientSetToContext runs; it was nil — the very ordering bug that produced the login panic")
	}
}
