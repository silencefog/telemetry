package main

import (
	"context"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	generated "github.com/silencefog/telemetry/server/generated"
)

var (
	validTokenCtx = contextWithToken()
)

func contextWithToken() context.Context {
	godotenv.Load()
	authToken := os.Getenv("AUTH_TOKEN")
	md := metadata.Pairs("authorization", "Bearer "+authToken)
	return metadata.NewIncomingContext(context.Background(), md)
}

func TestServer_StartsEmpty(t *testing.T) {
	s := &server{}

	assert.Empty(t, s.subscribers)
}

func TestPushTemperature_InvalidToken(t *testing.T) {
	s := &server{}

	req := &generated.TemperatureRequest{Value: 22.5, Timestamp: timestamppb.Now()}
	resp, err := s.PushTemperature(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestPushTemperature_ValidToken(t *testing.T) {
	s := &server{}

	req := &generated.TemperatureRequest{Value: 22.5, Timestamp: timestamppb.Now()}
	resp, err := s.PushTemperature(validTokenCtx, req)

	require.NoError(t, err)
	assert.IsType(t, &emptypb.Empty{}, resp)
}

func TestPushTemperature_SendsToSubscribers(t *testing.T) {
	s := &server{}

	mockSub := make(chan *generated.TemperatureResponse, 1)
	s.subscribers = append(s.subscribers, mockSub)

	req := &generated.TemperatureRequest{Value: 22.5, Timestamp: timestamppb.Now()}
	s.PushTemperature(validTokenCtx, req)

	data := <-mockSub
	assert.Equal(t, 22.5, data.Value)
}
