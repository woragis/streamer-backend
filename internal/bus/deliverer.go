package bus

import "context"

type Deliverer interface {
	Deliver(ctx context.Context, roomID, domain, eventType string, payload any)
}
