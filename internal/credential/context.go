package credential

import "context"

type policyKey struct{}

// WithPolicy attaches a Policy to ctx for providers to read during Fetch.
func WithPolicy(ctx context.Context, policy Policy) context.Context {
	return context.WithValue(ctx, policyKey{}, policy)
}

// PolicyFrom returns the Policy attached to ctx, or DefaultPolicy when absent.
func PolicyFrom(ctx context.Context) Policy {
	if ctx == nil {
		return DefaultPolicy
	}
	if p, ok := ctx.Value(policyKey{}).(Policy); ok {
		return p
	}
	return DefaultPolicy
}
