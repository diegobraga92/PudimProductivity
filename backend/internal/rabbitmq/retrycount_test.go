package rabbitmq

import (
"testing"

amqp "github.com/rabbitmq/amqp091-go"
)

func TestRetryCount(t *testing.T) {
if got := retryCount(nil); got != 0 {
t.Fatalf("nil headers: got %d", got)
}
if got := retryCount(amqp.Table{}); got != 0 {
t.Fatalf("empty headers: got %d", got)
}
h := amqp.Table{
"x-death": []interface{}{
amqp.Table{"count": int32(3), "reason": "rejected", "queue": "notifications"},
},
}
if got := retryCount(h); got != 3 {
t.Fatalf("expected 3, got %d", got)
}
}
