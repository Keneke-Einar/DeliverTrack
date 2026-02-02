package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/streadway/amqp"
)

// Event represents a message event
type Event struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	Timestamp int64                  `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	// Tracing context for distributed tracing
	TraceContext *TraceContext `json:"trace_context,omitempty"`
}

// TraceContext holds distributed tracing information
type TraceContext struct {
	TraceID      string `json:"trace_id"`
	SpanID       string `json:"span_id"`
	ParentSpanID string `json:"parent_span_id,omitempty"`
	ServiceName  string `json:"service_name"`
	Operation    string `json:"operation"`
}

// NewEventWithTrace creates a new event with tracing context
func NewEventWithTrace(eventType, source, operation string, data map[string]interface{}, traceCtx *TraceContext) Event {
	eventID := generateEventID()
	
	event := Event{
		ID:        eventID,
		Type:      eventType,
		Source:    source,
		Timestamp: time.Now().Unix(),
		Data:      data,
	}
	
	if traceCtx != nil {
		event.TraceContext = traceCtx
	}
	
	return event
}

// ExtractTraceContextFromContext extracts trace context from Go context
func ExtractTraceContextFromContext(ctx context.Context, serviceName, operation string) *TraceContext {
	// Check for trace context in context (you would implement this based on your tracing framework)
	traceID := getTraceIDFromContext(ctx)
	spanID := getSpanIDFromContext(ctx)
	parentSpanID := getParentSpanIDFromContext(ctx)
	
	if traceID == "" {
		// Generate new trace if none exists
		traceID = generateTraceID()
		spanID = generateSpanID()
	}
	
	return &TraceContext{
		TraceID:      traceID,
		SpanID:       spanID,
		ParentSpanID: parentSpanID,
		ServiceName:  serviceName,
		Operation:    operation,
	}
}

// ContextWithTraceContext adds trace context to Go context
func ContextWithTraceContext(ctx context.Context, traceCtx *TraceContext) context.Context {
	if traceCtx == nil {
		return ctx
	}
	
	// Add trace context to context (implement based on your tracing framework)
	ctx = context.WithValue(ctx, "trace_id", traceCtx.TraceID)
	ctx = context.WithValue(ctx, "span_id", traceCtx.SpanID)
	if traceCtx.ParentSpanID != "" {
		ctx = context.WithValue(ctx, "parent_span_id", traceCtx.ParentSpanID)
	}
	
	return ctx
}

// Helper functions (implement based on your tracing framework)
func getTraceIDFromContext(ctx context.Context) string {
	if val := ctx.Value("trace_id"); val != nil {
		return val.(string)
	}
	return ""
}

func getSpanIDFromContext(ctx context.Context) string {
	if val := ctx.Value("span_id"); val != nil {
		return val.(string)
	}
	return ""
}

func getParentSpanIDFromContext(ctx context.Context) string {
	if val := ctx.Value("parent_span_id"); val != nil {
		return val.(string)
	}
	return ""
}

func generateTraceID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func generateSpanID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func generateEventID() string {
	return fmt.Sprintf("evt_%d", time.Now().UnixNano())
}

// Publisher interface for publishing events
type Publisher interface {
	Publish(ctx context.Context, exchange, routingKey string, event Event) error
	Close() error
}

// Consumer interface for consuming events
type Consumer interface {
	Consume(queue string, handler func(Event) error) error
	Close() error
}

// RabbitMQPublisher implements Publisher interface
type RabbitMQPublisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

// NewRabbitMQPublisher creates a new RabbitMQ publisher
func NewRabbitMQPublisher(url string) (*RabbitMQPublisher, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	return &RabbitMQPublisher{
		conn:    conn,
		channel: channel,
	}, nil
}

// Publish publishes an event to RabbitMQ
func (p *RabbitMQPublisher) Publish(ctx context.Context, exchange, routingKey string, event Event) error {
	// Declare exchange if it doesn't exist
	err := p.channel.ExchangeDeclare(
		exchange, // name
		"topic",  // type
		true,     // durable
		false,    // auto-deleted
		false,    // internal
		false,    // no-wait
		nil,      // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Serialize event to JSON
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Publish message
	err = p.channel.Publish(
		exchange,   // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			Timestamp:    time.Now(),
			MessageId:    event.ID,
			CorrelationId: getCorrelationID(ctx),
		})
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.Printf("Published event %s to exchange %s with routing key %s", event.ID, exchange, routingKey)
	return nil
}

// Close closes the publisher
func (p *RabbitMQPublisher) Close() error {
	if p.channel != nil {
		p.channel.Close()
	}
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}

// RabbitMQConsumer implements Consumer interface
type RabbitMQConsumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

// NewRabbitMQConsumer creates a new RabbitMQ consumer
func NewRabbitMQConsumer(url string) (*RabbitMQConsumer, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	return &RabbitMQConsumer{
		conn:    conn,
		channel: channel,
	}, nil
}

// Consume starts consuming messages from a queue
func (c *RabbitMQConsumer) Consume(queue string, handler func(Event) error) error {
	// Declare queue
	_, err := c.channel.QueueDeclare(
		queue, // name
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		amqp.Table{
			"x-dead-letter-exchange": "dead-letter-exchange",
		}, // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Start consuming
	msgs, err := c.channel.Consume(
		queue, // queue
		"",    // consumer
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	go func() {
		for d := range msgs {
			var event Event
			if err := json.Unmarshal(d.Body, &event); err != nil {
				log.Printf("Failed to unmarshal event: %v", err)
				d.Nack(false, false) // Don't requeue
				continue
			}

			if err := handler(event); err != nil {
				log.Printf("Failed to handle event %s: %v", event.ID, err)
				d.Nack(false, false) // Don't requeue, send to dead letter queue
				continue
			}

			d.Ack(false)
		}
	}()

	log.Printf("Started consuming from queue: %s", queue)
	return nil
}

// Close closes the consumer
func (c *RabbitMQConsumer) Close() error {
	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// getCorrelationID extracts correlation ID from context
func getCorrelationID(ctx context.Context) string {
	if correlationID, ok := ctx.Value("correlation_id").(string); ok {
		return correlationID
	}
	return ""
}

// SetupDeadLetterExchange sets up dead letter exchange and queue
func SetupDeadLetterExchange(url string) error {
	conn, err := amqp.Dial(url)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	defer conn.Close()

	channel, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}
	defer channel.Close()

	// Declare dead letter exchange
	err = channel.ExchangeDeclare(
		"dead-letter-exchange", // name
		"topic",                // type
		true,                   // durable
		false,                  // auto-deleted
		false,                  // internal
		false,                  // no-wait
		nil,                    // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare dead letter exchange: %w", err)
	}

	// Declare dead letter queue
	_, err = channel.QueueDeclare(
		"dead-letter-queue", // name
		true,                // durable
		false,               // delete when unused
		false,               // exclusive
		false,               // no-wait
		nil,                 // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare dead letter queue: %w", err)
	}

	// Bind dead letter queue to exchange
	err = channel.QueueBind(
		"dead-letter-queue",    // queue name
		"#",                    // routing key
		"dead-letter-exchange", // exchange
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind dead letter queue: %w", err)
	}

	log.Println("Dead letter exchange and queue setup completed")
	return nil
}
