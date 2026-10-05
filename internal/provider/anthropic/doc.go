// Package anthropic translates between the neutral internal/provider types and
// the Anthropic Messages API wire format.
//
// It is the Messages-format wire adapter shared by the opencode and anthropic
// endpoints (ADR 0008). HTTP handling, headers, retries, and key handling live
// elsewhere.
package anthropic
