package adapters

import (
	"context"
	"strconv"

	"github.com/Keneke-Einar/delivertrack/internal/notification/domain"
	"github.com/Keneke-Einar/delivertrack/internal/notification/ports"
	notificationProto "github.com/Keneke-Einar/delivertrack/proto/notification"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCHandler handles gRPC requests for notification operations
type GRPCHandler struct {
	notificationProto.UnimplementedNotificationServiceServer
	service ports.NotificationService
}

// NewGRPCHandler creates a new gRPC handler
func NewGRPCHandler(service ports.NotificationService) *GRPCHandler {
	return &GRPCHandler{
		service: service,
	}
}

// SendNotification implements notification.NotificationServiceServer
func (h *GRPCHandler) SendNotification(ctx context.Context, req *notificationProto.SendNotificationRequest) (*notificationProto.SendNotificationResponse, error) {
	recipientID, err := strconv.Atoi(req.RecipientId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid recipient_id: %v", err)
	}

	// Map proto NotificationType to domain (assuming channel for simplicity)
	notifType := domain.NotificationType(req.Channel.String())

	notif, err := h.service.SendNotification(ctx, recipientID, notifType, req.Subject, req.Message, req.RecipientId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to send notification: %v", err)
	}

	resp := &notificationProto.SendNotificationResponse{
		NotificationId: strconv.Itoa(notif.ID),
		Status:         notificationProto.NotificationStatus_NOTIFICATION_STATUS_SENT,
		SentAt:         notif.CreatedAt.Unix(),
	}

	return resp, nil
}

// GetNotificationHistory implements notification.NotificationServiceServer
func (h *GRPCHandler) GetNotificationHistory(ctx context.Context, req *notificationProto.GetNotificationHistoryRequest) (*notificationProto.GetNotificationHistoryResponse, error) {
	recipientID, err := strconv.Atoi(req.RecipientId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid recipient_id: %v", err)
	}

	limit := 10
	if req.Pagination != nil && req.Pagination.PageSize > 0 {
		limit = int(req.Pagination.PageSize)
	}

	notifications, err := h.service.GetUserNotifications(ctx, recipientID, limit)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get notification history: %v", err)
	}

	var notifProtos []*notificationProto.Notification
	for _, n := range notifications {
		status := notificationProto.NotificationStatus_NOTIFICATION_STATUS_UNSPECIFIED
		switch n.Status {
		case "pending":
			status = notificationProto.NotificationStatus_NOTIFICATION_STATUS_PENDING
		case "sent":
			status = notificationProto.NotificationStatus_NOTIFICATION_STATUS_SENT
		case "failed":
			status = notificationProto.NotificationStatus_NOTIFICATION_STATUS_FAILED
		}

		notifProtos = append(notifProtos, &notificationProto.Notification{
			NotificationId: strconv.Itoa(n.ID),
			RecipientId:    strconv.Itoa(n.UserID),
			Type:           notificationProto.NotificationType_NOTIFICATION_TYPE_SYSTEM_ALERT, // default
			Channel:        notificationProto.NotificationChannel_CHANNEL_EMAIL, // default
			Subject:        n.Subject,
			Message:        n.Message,
			Status:         status,
			Read:           false,
			CreatedAt:      n.CreatedAt.Unix(),
			SentAt:         n.CreatedAt.Unix(),
			ReadAt:         0,
		})
	}

	return &notificationProto.GetNotificationHistoryResponse{
		Notifications: notifProtos,
	}, nil
}

// MarkAsRead implements notification.NotificationServiceServer
func (h *GRPCHandler) MarkAsRead(ctx context.Context, req *notificationProto.MarkAsReadRequest) (*notificationProto.MarkAsReadResponse, error) {
	notificationID, err := strconv.Atoi(req.NotificationId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid notification_id: %v", err)
	}

	err = h.service.MarkAsRead(ctx, notificationID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to mark as read: %v", err)
	}

	return &notificationProto.MarkAsReadResponse{}, nil
}

// SendBulkNotifications implements notification.NotificationServiceServer
func (h *GRPCHandler) SendBulkNotifications(ctx context.Context, req *notificationProto.SendBulkNotificationsRequest) (*notificationProto.SendBulkNotificationsResponse, error) {
	// TODO: Implement when service supports bulk
	return nil, status.Errorf(codes.Unimplemented, "method SendBulkNotifications not implemented")
}

// UpdatePreferences implements notification.NotificationServiceServer
func (h *GRPCHandler) UpdatePreferences(ctx context.Context, req *notificationProto.UpdatePreferencesRequest) (*notificationProto.UpdatePreferencesResponse, error) {
	// TODO: Implement when service supports preferences
	return nil, status.Errorf(codes.Unimplemented, "method UpdatePreferences not implemented")
}

// GetPreferences implements notification.NotificationServiceServer
func (h *GRPCHandler) GetPreferences(ctx context.Context, req *notificationProto.GetPreferencesRequest) (*notificationProto.GetPreferencesResponse, error) {
	// TODO: Implement when service supports preferences
	return nil, status.Errorf(codes.Unimplemented, "method GetPreferences not implemented")
}

// Subscribe implements notification.NotificationServiceServer
func (h *GRPCHandler) Subscribe(req *notificationProto.SubscribeRequest, stream notificationProto.NotificationService_SubscribeServer) error {
	// TODO: Implement streaming
	return status.Errorf(codes.Unimplemented, "method Subscribe not implemented")
}

// SendDeliveryUpdate implements notification.NotificationServiceServer
func (h *GRPCHandler) SendDeliveryUpdate(ctx context.Context, req *notificationProto.SendDeliveryUpdateRequest) (*notificationProto.SendDeliveryUpdateResponse, error) {
	// TODO: Implement when service supports delivery updates
	return nil, status.Errorf(codes.Unimplemented, "method SendDeliveryUpdate not implemented")
}
