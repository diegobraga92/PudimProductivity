// Package sharedkernel holds the domain primitives that every bounded context
// (audit, collaboration, content, insights, productivity) agrees on — the
// "shared kernel" of the DDD model. Contexts depend on these types but never on
// each other's internals.
//
// The shared kernel is intentionally small. It currently provides the ID value
// object used to identify entities; anything that is not needed by more than
// one context should live inside that context instead.
package sharedkernel
