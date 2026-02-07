package adapters

import (
	"context"
	"strconv"

	"github.com/Keneke-Einar/delivertrack/internal/delivery/ports"
	"github.com/Keneke-Einar/delivertrack/pkg/auth/domain"
	"github.com/Keneke-Einar/delivertrack/pkg/grpcinterceptors"
	"github.com/Keneke-Einar/delivertrack/proto/common"
	deliveryProto "github.com/Keneke-Einar/delivertrack/proto/delivery"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCHandler handles gRPC requests for delivery operations
type GRPCHandler struct {
	deliveryProto.UnimplementedDeliveryServiceServer
	service ports.DeliveryService
}

// NewGRPCHandler creates a new gRPC handler
func NewGRPCHandler(service ports.DeliveryService) *GRPCHandler {
	return &GRPCHandler{
		service: service,
	}
}

// CreateDelivery implements delivery.DeliveryServiceServer
func (h *GRPCHandler) CreateDelivery(ctx context.Context, req *deliveryProto.CreateDeliveryRequest) (*deliveryProto.CreateDeliveryResponse, error) {
	// Parse customer_id
	customerID, err := strconv.Atoi(req.CustomerId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid customer_id: %v", err)
	}

	// Map proto request to service request
	serviceReq := ports.CreateDeliveryRequest{
		CustomerID:       customerID,
		PickupLocation:   req.PickupLocation.Address, // Assuming address is used
		DeliveryLocation: req.DeliveryLocation.Address,
		Notes:            req.SpecialInstructions,
	}

	// Call service
	delivery, err := h.service.CreateDelivery(ctx, serviceReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create delivery: %v", err)
	}

	// Map response
	resp := &deliveryProto.CreateDeliveryResponse{
		DeliveryId:     strconv.Itoa(delivery.ID),
		TrackingNumber: "", // TODO: generate tracking number
		CreatedAt:      delivery.CreatedAt.Unix(),
	}

	return resp, nil
}

// GetDelivery implements delivery.DeliveryServiceServer
func (h *GRPCHandler) GetDelivery(ctx context.Context, req *deliveryProto.GetDeliveryRequest) (*deliveryProto.GetDeliveryResponse, error) {
	deliveryID, err := strconv.Atoi(req.DeliveryId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid delivery_id: %v", err)
	}

	serviceReq := ports.GetDeliveryRequest{
		ID: deliveryID,
	}

	// Extract user claims from context
	if claims, ok := ctx.Value(grpcinterceptors.UserClaimsContextKey).(*domain.Claims); ok {
		serviceReq.Role = claims.Role
		serviceReq.UserCustomerID = claims.CustomerID
		serviceReq.UserCourierID = claims.CourierID
	}

	d, err := h.service.GetDelivery(ctx, serviceReq)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "delivery not found: %v", err)
	}

	resp := &deliveryProto.GetDeliveryResponse{
		Delivery: &deliveryProto.Delivery{
			DeliveryId:       strconv.Itoa(d.ID),
			CustomerId:       strconv.Itoa(d.CustomerID),
			DriverId:         strconv.Itoa(*d.CourierID),
			PickupLocation:   &common.Location{Address: d.PickupLocation},
			DeliveryLocation: &common.Location{Address: d.DeliveryLocation},
			Status:           deliveryProto.DeliveryStatus(deliveryProto.DeliveryStatus_value[d.Status]),
			CreatedAt:        d.CreatedAt.Unix(),
			UpdatedAt:        d.UpdatedAt.Unix(),
		},
	}

	return resp, nil
}

// UpdateDeliveryStatus implements delivery.DeliveryServiceServer
func (h *GRPCHandler) UpdateDeliveryStatus(ctx context.Context, req *deliveryProto.UpdateDeliveryStatusRequest) (*deliveryProto.UpdateDeliveryStatusResponse, error) {
	deliveryID, err := strconv.Atoi(req.DeliveryId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid delivery_id: %v", err)
	}

	serviceReq := ports.UpdateDeliveryStatusRequest{
		ID:     deliveryID,
		Status: req.Status.String(), // Convert enum to string
		Notes:  req.Notes,
	}

	err = h.service.UpdateDeliveryStatus(ctx, serviceReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update delivery status: %v", err)
	}

	return &deliveryProto.UpdateDeliveryStatusResponse{}, nil
}

// ListDeliveries implements delivery.DeliveryServiceServer
func (h *GRPCHandler) ListDeliveries(ctx context.Context, req *deliveryProto.ListDeliveriesRequest) (*deliveryProto.ListDeliveriesResponse, error) {
	customerID, err := strconv.Atoi(req.CustomerId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid customer_id: %v", err)
	}

	serviceReq := ports.ListDeliveriesRequest{
		Status:     req.Status.String(),
		CustomerID: customerID,
	}

	deliveries, err := h.service.ListDeliveries(ctx, serviceReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list deliveries: %v", err)
	}

	var deliveryProtos []*deliveryProto.Delivery
	for _, d := range deliveries {
		deliveryProtos = append(deliveryProtos, &deliveryProto.Delivery{
			DeliveryId:       strconv.Itoa(d.ID),
			CustomerId:       strconv.Itoa(d.CustomerID),
			DriverId:         strconv.Itoa(*d.CourierID),
			PickupLocation:   &common.Location{Address: d.PickupLocation},
			DeliveryLocation: &common.Location{Address: d.DeliveryLocation},
			Status:           deliveryProto.DeliveryStatus(deliveryProto.DeliveryStatus_value[d.Status]),
			CreatedAt:        d.CreatedAt.Unix(),
			UpdatedAt:        d.UpdatedAt.Unix(),
		})
	}

	return &deliveryProto.ListDeliveriesResponse{
		Deliveries: deliveryProtos,
	}, nil
}

// AssignDriver implements delivery.DeliveryServiceServer
func (h *GRPCHandler) AssignDriver(ctx context.Context, req *deliveryProto.AssignDriverRequest) (*deliveryProto.AssignDriverResponse, error) {
	// TODO: Implement when service supports it
	return nil, status.Errorf(codes.Unimplemented, "method AssignDriver not implemented")
}

// CancelDelivery implements delivery.DeliveryServiceServer
func (h *GRPCHandler) CancelDelivery(ctx context.Context, req *deliveryProto.CancelDeliveryRequest) (*deliveryProto.CancelDeliveryResponse, error) {
	// TODO: Implement when service supports it
	return nil, status.Errorf(codes.Unimplemented, "method CancelDelivery not implemented")
}

// GetDriverDeliveries implements delivery.DeliveryServiceServer
func (h *GRPCHandler) GetDriverDeliveries(ctx context.Context, req *deliveryProto.GetDriverDeliveriesRequest) (*deliveryProto.GetDriverDeliveriesResponse, error) {
	// TODO: Implement when service supports it
	return nil, status.Errorf(codes.Unimplemented, "method GetDriverDeliveries not implemented")
}

// OptimizeRoute implements delivery.DeliveryServiceServer
func (h *GRPCHandler) OptimizeRoute(ctx context.Context, req *deliveryProto.OptimizeRouteRequest) (*deliveryProto.OptimizeRouteResponse, error) {
	// TODO: Implement when service supports it
	return nil, status.Errorf(codes.Unimplemented, "method OptimizeRoute not implemented")
}

// ConfirmDelivery implements delivery.DeliveryServiceServer
func (h *GRPCHandler) ConfirmDelivery(ctx context.Context, req *deliveryProto.ConfirmDeliveryRequest) (*deliveryProto.ConfirmDeliveryResponse, error) {
	// TODO: Implement when service supports it
	return nil, status.Errorf(codes.Unimplemented, "method ConfirmDelivery not implemented")
}
