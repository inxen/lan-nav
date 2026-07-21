package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestHTTPClient_Internal 验证 newHTTPClient 的 blockInternal 开关：
//   - false（管理后台抓取）：允许内网/回环地址；
//   - true（公开 favicon）：拒绝内网/回环地址。
func TestHTTPClient_Internal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// 公开接口：应拒绝内网/回环（httptest 默认绑定 127.0.0.1，属阻断段）
	pub := newHTTPClient(5*time.Second, true)
	if _, err := pub.Get(srv.URL); err == nil {
		t.Fatal("blockInternal=true 应拒绝内网/回环地址，但未报错")
	}

	// 管理后台：应允许内网/回环
	admin := newHTTPClient(5*time.Second, false)
	resp, err := admin.Get(srv.URL)
	if err != nil {
		t.Fatalf("blockInternal=false 应允许内网/回环地址，但报错: %v", err)
	}
	resp.Body.Close()
}
