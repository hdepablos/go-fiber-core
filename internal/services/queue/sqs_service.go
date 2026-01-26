package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

// SQSService encapsula las operaciones de SQS
type SQSService struct {
	client   *sqs.Client
	queueURL string
}

// Message representa un mensaje de SQS
type Message struct {
	ID      string                 `json:"id"`
	Body    string                 `json:"body"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Source  string                 `json:"source"`
	Created string                 `json:"created"`
}

// NewSQSService crea una nueva instancia del servicio SQS.
// Ahora recibe el cliente de SQS y la URL de la cola como dependencias.
// Esto elimina la lógica de configuración duplicada.
func NewSQSService(client *sqs.Client, queueURL string) *SQSService {
	return &SQSService{
		client:   client,
		queueURL: queueURL,
	}
}

// SendMessage envía un mensaje a la cola SQS
func (s *SQSService) SendMessage(ctx context.Context, message *Message) error {
	messageBody, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("error marshaling message: %w", err)
	}

	input := &sqs.SendMessageInput{
		QueueUrl:    aws.String(s.queueURL),
		MessageBody: aws.String(string(messageBody)),
		MessageAttributes: map[string]types.MessageAttributeValue{
			"Source": {
				DataType:    aws.String("String"),
				StringValue: aws.String(message.Source),
			},
		},
	}

	result, err := s.client.SendMessage(ctx, input)
	if err != nil {
		return fmt.Errorf("error sending message: %w", err)
	}

	log.Printf("✅ Message sent successfully --- MessageId: %s", *result.MessageId)
	return nil
}

// ReceiveMessages recibe mensajes de la cola SQS
func (s *SQSService) ReceiveMessages(ctx context.Context, maxMessages int32) ([]types.Message, error) {
	if maxMessages <= 0 {
		maxMessages = 10
	}

	input := &sqs.ReceiveMessageInput{
		QueueUrl:              aws.String(s.queueURL),
		MaxNumberOfMessages:   maxMessages,
		WaitTimeSeconds:       20, // Long polling
		MessageAttributeNames: []string{"All"},
	}

	result, err := s.client.ReceiveMessage(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("error receiving messages: %w", err)
	}

	log.Printf("📨 Received %d messages", len(result.Messages))
	return result.Messages, nil
}

// DeleteMessage elimina un mensaje de la cola SQS
func (s *SQSService) DeleteMessage(ctx context.Context, receiptHandle string) error {
	input := &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(s.queueURL),
		ReceiptHandle: aws.String(receiptHandle),
	}

	_, err := s.client.DeleteMessage(ctx, input)
	if err != nil {
		return fmt.Errorf("error deleting message: %w", err)
	}

	log.Printf("🗑️  Message deleted successfully")
	return nil
}

// ProcessMessage procesa un mensaje específico
func (s *SQSService) ProcessMessage(ctx context.Context, sqsMessage types.Message) error {
	var message Message
	if err := json.Unmarshal([]byte(*sqsMessage.Body), &message); err != nil {
		log.Printf("❌ Error unmarshaling message: %v", err)
		return fmt.Errorf("intentional DLQ trigger")
	}

	log.Printf("🔄 Processing message: %s from %s", message.ID, message.Source)

	log.Printf("📋 Message content: %s", message.Body)
	if message.Data != nil {
		log.Printf("📊 Message data: %+v", message.Data)
	}

	return nil
}

// GetQueueAttributes obtiene los atributos de la cola
func (s *SQSService) GetQueueAttributes(ctx context.Context) (map[string]string, error) {
	input := &sqs.GetQueueAttributesInput{
		QueueUrl: aws.String(s.queueURL),
		AttributeNames: []types.QueueAttributeName{
			types.QueueAttributeNameApproximateNumberOfMessages,
			types.QueueAttributeNameApproximateNumberOfMessagesNotVisible,
		},
	}

	result, err := s.client.GetQueueAttributes(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("error getting queue attributes: %w", err)
	}

	return result.Attributes, nil
}
