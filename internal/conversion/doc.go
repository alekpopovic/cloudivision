// Package conversion contains pure, in-process CRD version conversion building
// blocks. It intentionally does not register a webhook server or mutate CRD
// conversion settings. When a second API version exists, version types act as
// spokes and convert through one internal hub representation; conversions must
// remain deterministic, side-effect free and round-trip tested.
package conversion
