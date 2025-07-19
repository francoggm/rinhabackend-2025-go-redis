package processors

import "context"

type Processor interface {
	ProcessEvent(ctx context.Context, event any) error
}
