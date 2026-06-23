package gateway

import (
	"context"
	"errors"
	"testing"

	"github.com/UpCloudLtd/upcloud-go-api/v8/upcloud"
	"github.com/UpCloudLtd/upcloud-go-api/v8/upcloud/request"
)

type mockConnections struct {
	conns []upcloud.GatewayConnection
	err   error
}

func (m *mockConnections) GetGatewayConnections(_ context.Context, _ *request.GetGatewayConnectionsRequest) ([]upcloud.GatewayConnection, error) {
	return m.conns, m.err
}

type mockTunnels struct {
	conns []upcloud.GatewayConnection
	tuns  []upcloud.GatewayTunnel
	err   error
}

func (m *mockTunnels) GetGatewayConnections(_ context.Context, _ *request.GetGatewayConnectionsRequest) ([]upcloud.GatewayConnection, error) {
	return m.conns, m.err
}

func (m *mockTunnels) GetGatewayConnectionTunnels(_ context.Context, _ *request.GetGatewayConnectionTunnelsRequest) ([]upcloud.GatewayTunnel, error) {
	return m.tuns, m.err
}

func TestParseConnectionID_V1_UUID(t *testing.T) {
	svcUUID := "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
	connUUID := "11111111-2222-3333-4444-555555555555"
	id := svcUUID + "/" + connUUID

	resultSvc, resultConn, err := parseConnectionID(context.Background(), &mockConnections{}, id, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resultSvc != svcUUID {
		t.Errorf("expected service UUID %q, got %q", svcUUID, resultSvc)
	}
	if resultConn != connUUID {
		t.Errorf("expected connection UUID %q, got %q", connUUID, resultConn)
	}
}

func TestParseConnectionID_V0_BareName(t *testing.T) {
	svcUUID := "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
	connName := "my-connection"
	resolvedUUID := "11111111-2222-3333-4444-555555555555"

	mock := &mockConnections{
		conns: []upcloud.GatewayConnection{
			{Name: connName, UUID: resolvedUUID},
		},
	}

	resultSvc, resultConn, err := parseConnectionID(context.Background(), mock, connName, svcUUID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resultSvc != svcUUID {
		t.Errorf("expected service UUID %q, got %q", svcUUID, resultSvc)
	}
	if resultConn != resolvedUUID {
		t.Errorf("expected resolved UUID %q, got %q", resolvedUUID, resultConn)
	}
}

func TestParseConnectionID_V0_NameNotFound(t *testing.T) {
	mock := &mockConnections{
		conns: []upcloud.GatewayConnection{
			{Name: "other-connection", UUID: "11111111-2222-3333-4444-555555555555"},
		},
	}

	_, _, err := parseConnectionID(context.Background(), mock, "nonexistent", "a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	if err == nil {
		t.Fatal("expected error for nonexistent connection name")
	}
}

func TestParseConnectionID_Invalid(t *testing.T) {
	_, _, err := parseConnectionID(context.Background(), &mockConnections{}, "bare-name", "")
	if err == nil {
		t.Fatal("expected error when no gateway in state")
	}
}

func TestParseTunnelID_V1_UUIDs(t *testing.T) {
	svcUUID := "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
	connUUID := "11111111-2222-3333-4444-555555555555"
	tunUUID := "66666666-7777-8888-9999-aaaaaaaaaaaa"
	id := svcUUID + "/" + connUUID + "/" + tunUUID

	resultSvc, resultConn, resultTun, err := parseTunnelID(context.Background(), &mockTunnels{}, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resultSvc != svcUUID {
		t.Errorf("expected service UUID %q, got %q", svcUUID, resultSvc)
	}
	if resultConn != connUUID {
		t.Errorf("expected connection UUID %q, got %q", connUUID, resultConn)
	}
	if resultTun != tunUUID {
		t.Errorf("expected tunnel UUID %q, got %q", tunUUID, resultTun)
	}
}

func TestParseTunnelID_V0_Names(t *testing.T) {
	svcUUID := "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
	connName := "my-connection"
	connUUID := "11111111-2222-3333-4444-555555555555"
	tunName := "test-tunnel"
	tunUUID := "66666666-7777-8888-9999-aaaaaaaaaaaa"
	id := svcUUID + "/" + connName + "/" + tunName

	mock := &mockTunnels{
		conns: []upcloud.GatewayConnection{
			{Name: connName, UUID: connUUID},
		},
		tuns: []upcloud.GatewayTunnel{
			{Name: tunName, UUID: tunUUID},
		},
	}

	resultSvc, resultConn, resultTun, err := parseTunnelID(context.Background(), mock, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resultSvc != svcUUID {
		t.Errorf("expected service UUID %q, got %q", svcUUID, resultSvc)
	}
	if resultConn != connUUID {
		t.Errorf("expected resolved connection UUID %q, got %q", connUUID, resultConn)
	}
	if resultTun != tunUUID {
		t.Errorf("expected resolved tunnel UUID %q, got %q", tunUUID, resultTun)
	}
}

func TestParseTunnelID_V0_ConnectionNotFound(t *testing.T) {
	id := "a1b2c3d4-e5f6-7890-abcd-ef1234567890/nonexistent-conn/test-tunnel"

	mock := &mockTunnels{
		conns: []upcloud.GatewayConnection{
			{Name: "other-connection", UUID: "11111111-2222-3333-4444-555555555555"},
		},
	}

	_, _, _, err := parseTunnelID(context.Background(), mock, id)
	if err == nil {
		t.Fatal("expected error for nonexistent connection name")
	}
}

func TestParseTunnelID_V0_TunnelNotFound(t *testing.T) {
	svcUUID := "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
	connName := "my-connection"
	connUUID := "11111111-2222-3333-4444-555555555555"
	id := svcUUID + "/" + connName + "/nonexistent-tunnel"

	mock := &mockTunnels{
		conns: []upcloud.GatewayConnection{
			{Name: connName, UUID: connUUID},
		},
		tuns: []upcloud.GatewayTunnel{
			{Name: "other-tunnel", UUID: "66666666-7777-8888-9999-aaaaaaaaaaaa"},
		},
	}

	_, _, _, err := parseTunnelID(context.Background(), mock, id)
	if err == nil {
		t.Fatal("expected error for nonexistent tunnel name")
	}
}

func TestParseTunnelID_APIError(t *testing.T) {
	mock := &mockTunnels{
		err: errors.New("api failure"),
	}

	_, _, _, err := parseTunnelID(context.Background(), mock, "a1b2c3d4-e5f6-7890-abcd-ef1234567890/my-connection/test-tunnel")
	if err == nil {
		t.Fatal("expected error from API failure")
	}
}

func TestParseTunnelID_InvalidFormat(t *testing.T) {
	_, _, _, err := parseTunnelID(context.Background(), &mockTunnels{}, "not-enough-parts")
	if err == nil {
		t.Fatal("expected error for invalid ID format")
	}
}
