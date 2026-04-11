package feishu

import "testing"

func TestWSPoolSingleton(t *testing.T) {
	p1 := GetWSPool(3, 30)
	p2 := GetWSPool(3, 30)
	if p1 != p2 {
		t.Error("pool should be singleton")
	}
}

func TestWSPoolStatusDisconnected(t *testing.T) {
	p := GetWSPool(3, 30)
	if p.Status("nonexistent") != WSDisconnected {
		t.Error("nonexistent app should be disconnected")
	}
}

func TestNewWSConn(t *testing.T) {
	conn := NewWSConn("test-id", "test-secret", 3, 30)
	if conn.AppID != "test-id" {
		t.Errorf("expected test-id, got %s", conn.AppID)
	}
	if conn.Status != WSDisconnected {
		t.Error("initial status should be disconnected")
	}
}