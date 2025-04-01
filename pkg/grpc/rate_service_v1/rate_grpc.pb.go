// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v3.12.4
// source: rate.proto

package rate_service_v1

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	RateService_GetRates_FullMethodName    = "/rate_service.v1.RateService/GetRates"
	RateService_HealthCheck_FullMethodName = "/rate_service.v1.RateService/HealthCheck"
)

// RateServiceClient is the client API for RateService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type RateServiceClient interface {
	GetRates(ctx context.Context, in *GetRatesRequest, opts ...grpc.CallOption) (*GetRatesResponse, error)
	HealthCheck(ctx context.Context, in *HealthCheckRequest, opts ...grpc.CallOption) (*HealthCheckResponse, error)
}

type rateServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewRateServiceClient(cc grpc.ClientConnInterface) RateServiceClient {
	return &rateServiceClient{cc}
}

func (c *rateServiceClient) GetRates(ctx context.Context, in *GetRatesRequest, opts ...grpc.CallOption) (*GetRatesResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetRatesResponse)
	err := c.cc.Invoke(ctx, RateService_GetRates_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *rateServiceClient) HealthCheck(ctx context.Context, in *HealthCheckRequest, opts ...grpc.CallOption) (*HealthCheckResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(HealthCheckResponse)
	err := c.cc.Invoke(ctx, RateService_HealthCheck_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// RateServiceServer is the server API for RateService service.
// All implementations must embed UnimplementedRateServiceServer
// for forward compatibility.
type RateServiceServer interface {
	GetRates(context.Context, *GetRatesRequest) (*GetRatesResponse, error)
	HealthCheck(context.Context, *HealthCheckRequest) (*HealthCheckResponse, error)
	mustEmbedUnimplementedRateServiceServer()
}

// UnimplementedRateServiceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedRateServiceServer struct{}

func (UnimplementedRateServiceServer) GetRates(context.Context, *GetRatesRequest) (*GetRatesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetRates not implemented")
}
func (UnimplementedRateServiceServer) HealthCheck(context.Context, *HealthCheckRequest) (*HealthCheckResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method HealthCheck not implemented")
}
func (UnimplementedRateServiceServer) mustEmbedUnimplementedRateServiceServer() {}
func (UnimplementedRateServiceServer) testEmbeddedByValue()                     {}

// UnsafeRateServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to RateServiceServer will
// result in compilation errors.
type UnsafeRateServiceServer interface {
	mustEmbedUnimplementedRateServiceServer()
}

func RegisterRateServiceServer(s grpc.ServiceRegistrar, srv RateServiceServer) {
	// If the following call pancis, it indicates UnimplementedRateServiceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&RateService_ServiceDesc, srv)
}

func _RateService_GetRates_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetRatesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RateServiceServer).GetRates(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RateService_GetRates_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RateServiceServer).GetRates(ctx, req.(*GetRatesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RateService_HealthCheck_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(HealthCheckRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RateServiceServer).HealthCheck(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RateService_HealthCheck_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RateServiceServer).HealthCheck(ctx, req.(*HealthCheckRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// RateService_ServiceDesc is the grpc.ServiceDesc for RateService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var RateService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "rate_service.v1.RateService",
	HandlerType: (*RateServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetRates",
			Handler:    _RateService_GetRates_Handler,
		},
		{
			MethodName: "HealthCheck",
			Handler:    _RateService_HealthCheck_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "rate.proto",
}
