package controller

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func TestPendingRegistration_StoresOnlyPasswordHash(t *testing.T) {
	const password = "S3cret-Passw0rd"
	p, err := newPendingRegistration("20240001", "alice", password, "alice@example.com", bcrypt.MinCost, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(data), password) {
		t.Fatalf("serialized pending registration must not contain plaintext password: %s", data)
	}
	if strings.Contains(string(data), `"password"`) {
		t.Fatalf("serialized pending registration must not contain a plaintext password field: %s", data)
	}

	var decoded pendingRegistration
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !decoded.valid(time.Now()) {
		t.Fatal("decoded pending registration should be valid")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(decoded.PasswordHash), []byte(password)); err != nil {
		t.Fatalf("stored hash should verify the original password: %v", err)
	}
}

func TestPendingRegistration_RejectsIncompleteOrLegacyData(t *testing.T) {
	future := time.Now().Add(time.Hour).Format(time.RFC3339)
	cases := map[string]string{
		// 旧版 ResendVerificationEmail 写入的数据只有 email，会创建空学号/空密码用户
		"email only": `{"email":"a@example.com","expires_at":"` + future + `"}`,
		// 旧版 Register 写入的明文密码数据
		"legacy plaintext": `{"student_id":"20240001","username":"alice","password":"123456abc","email":"a@example.com","expires_at":"` + future + `"}`,
		"bad hash":         `{"student_id":"20240001","username":"alice","password_hash":"not-a-hash","email":"a@example.com","expires_at":"` + future + `"}`,
	}
	for name, raw := range cases {
		var p pendingRegistration
		if err := json.Unmarshal([]byte(raw), &p); err != nil {
			t.Fatalf("%s: unmarshal: %v", name, err)
		}
		if p.valid(time.Now()) {
			t.Errorf("%s: should be invalid", name)
		}
	}
}

func TestPendingRegistration_Expired(t *testing.T) {
	p, err := newPendingRegistration("20240001", "alice", "S3cret-Passw0rd", "alice@example.com", bcrypt.MinCost, time.Now().Add(-time.Minute))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.valid(time.Now()) {
		t.Fatal("expired pending registration should be invalid")
	}
}

func TestMaskEmail(t *testing.T) {
	if got := maskEmail("alice@example.com"); got != "al***@example.com" {
		t.Fatalf("unexpected mask: %s", got)
	}
	if got := maskEmail("ab@example.com"); got != "**@example.com" {
		t.Fatalf("unexpected mask: %s", got)
	}
}
