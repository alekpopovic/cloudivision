package logstore

import "context"

type ObjectStore struct{}
type LokiStore struct{}

func (ObjectStore) Append(context.Context, AppendLogRequest) error { return ErrNotImplemented }
func (ObjectStore) Read(context.Context, ReadLogRequest) (*ReadLogResult, error) {
	return nil, ErrNotImplemented
}
func (ObjectStore) Stream(context.Context, StreamLogRequest) (<-chan LogLine, error) {
	return nil, ErrNotImplemented
}
func (ObjectStore) Delete(context.Context, DeleteLogRequest) error { return ErrNotImplemented }

func (LokiStore) Append(context.Context, AppendLogRequest) error { return ErrNotImplemented }
func (LokiStore) Read(context.Context, ReadLogRequest) (*ReadLogResult, error) {
	return nil, ErrNotImplemented
}
func (LokiStore) Stream(context.Context, StreamLogRequest) (<-chan LogLine, error) {
	return nil, ErrNotImplemented
}
func (LokiStore) Delete(context.Context, DeleteLogRequest) error { return ErrNotImplemented }
