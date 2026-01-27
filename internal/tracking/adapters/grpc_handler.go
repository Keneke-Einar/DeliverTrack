package adapters

import (
	"context"
	"strconv"

	"github.com/Keneke-Einar/delivertrack/internal/tracking/ports"
	"github.com/Keneke-Einar/delivertrack/proto/common"
	trackingProto "github.com/Keneke-Einar/delivertrack/proto/tracking"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCHandler handles gRPC requests for tracking operations
type GRPCHandler struct {
	trackingProto.UnimplementedTrackingServiceServer
	service ports.TrackingService
}

// NewGRPCHandler creates a new gRPC handler
func NewGRPCHandler(service ports.TrackingService) *GRPCHandler {
	return &GRPCHandler{
		service: service,
	}
}

// CreateTracking implements tracking.TrackingServiceServer
func (h *GRPCHandler) CreateTracking(ctx context.Context, req *trackingProto.CreateTrackingRequest) (*trackingProto.CreateTrackingResponse, error) {
	// TODO: Implement when service supports it
	return nil, status.Errorf(codes.Unimplemented, "method CreateTracking not implemented")
}

// GetTracking implements tracking.TrackingServiceServer
func (h *GRPCHandler) GetTracking(ctx context.Context, req *trackingProto.GetTrackingRequest) (*trackingProto.GetTrackingResponse, error) {
	// TODO: Implement when service supports it
	return nil, status.Errorf(codes.Unimplemented, "method GetTracking not implemented")
}

// UpdateLocation implements tracking.TrackingServiceServer
func (h *GRPCHandler) UpdateLocation(ctx context.Context, req *trackingProto.UpdateLocationRequest) (*trackingProto.UpdateLocationResponse, error) {
	// For now, assume tracking_number is delivery ID
	deliveryID, err := strconv.Atoi(req.TrackingNumber)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tracking_number: %v", err)
	}

	serviceReq := ports.RecordLocationRequest{
		DeliveryID: deliveryID,
		CourierID:  0, // TODO: get from context or something
		Latitude:   req.Location.Latitude,
		Longitude:  req.Location.Longitude,
		Accuracy:   nil,
		Speed:      nil,
		Heading:    nil,
		Altitude:   nil,
	}

	location, err := h.service.RecordLocation(ctx, serviceReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to record location: %v", err)
	}

	resp := &trackingProto.UpdateLocationResponse{
		Success:   true,
		UpdatedAt: location.Timestamp.Unix(),
	}

	return resp, nil
}

// AddTrackingEvent implements tracking.TrackingServiceServer
func (h *GRPCHandler) AddTrackingEvent(ctx context.Context, req *trackingProto.AddTrackingEventRequest) (*trackingProto.AddTrackingEventResponse, error) {
	// TODO: Implement when service supports it
	return nil, status.Errorf(codes.Unimplemented, "method AddTrackingEvent not implemented")
}

// GetTrackingHistory implements tracking.TrackingServiceServer
func (h *GRPCHandler) GetTrackingHistory(ctx context.Context, req *trackingProto.GetTrackingHistoryRequest) (*trackingProto.GetTrackingHistoryResponse, error) {
	deliveryID, err := strconv.Atoi(req.TrackingNumber)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tracking_number: %v", err)
	}

	serviceReq := ports.GetDeliveryTrackRequest{
		DeliveryID: deliveryID,
		Limit:      100, // Default limit
	}

	locations, err := h.service.GetDeliveryTrack(ctx, serviceReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get tracking history: %v", err)
	}

	var locationUpdates []*trackingProto.LocationUpdate
	for _, loc := range locations {
		locationUpdates = append(locationUpdates, &trackingProto.LocationUpdate{
			TrackingNumber: req.TrackingNumber,
			Location: &common.Location{
				Latitude:  loc.Latitude,
				Longitude: loc.Longitude,
			},
			Timestamp: loc.Timestamp.Unix(),
		})
	}

	return &trackingProto.GetTrackingHistoryResponse{
		Events: []*trackingProto.TrackingEvent{}, // TODO: map locations to events
	}, nil
}

// StreamLocation implements tracking.TrackingServiceServer
func (h *GRPCHandler) StreamLocation(req *trackingProto.StreamLocationRequest, stream trackingProto.TrackingService_StreamLocationServer) error {
	// TODO: Implement streaming
	return status.Errorf(codes.Unimplemented, "method StreamLocation not implemented")
}

// BatchUpdateLocations implements tracking.TrackingServiceServer
func (h *GRPCHandler) BatchUpdateLocations(ctx context.Context, req *trackingProto.BatchUpdateLocationsRequest) (*trackingProto.BatchUpdateLocationsResponse, error) {
	// TODO: Implement batch updates
	return nil, status.Errorf(codes.Unimplemented, "method BatchUpdateLocations not implemented")
}
