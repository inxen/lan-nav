package main

import (
	"net/http"
	"testing"
)

// newReq 构造一个合成请求，指定 TCP 对端地址与可选转发头
func newReq(remoteAddr, xff, xri string) *http.Request {
	req := &http.Request{
		RemoteAddr: remoteAddr,
		Header:     make(http.Header),
	}
	if xff != "" {
		req.Header.Set("X-Forwarded-For", xff)
	}
	if xri != "" {
		req.Header.Set("X-Real-IP", xri)
	}
	return req
}

// 公网直连（对端非可信代理）携带伪造 XFF：必须忽略，返回真实对端 IP
func TestClientIP_PublicDirectForgeryIgnored(t *testing.T) {
	req := newReq("203.0.113.5:54321", "192.168.1.5", "")
	if got := clientIP(req); got != "203.0.113.5" {
		t.Fatalf("公网直连伪造 XFF 未被忽略: got=%q want=%q", got, "203.0.113.5")
	}
}

// 公网直连携带伪造 X-Real-IP：同样必须忽略
func TestClientIP_PublicDirectXRealIPIgnored(t *testing.T) {
	req := newReq("203.0.113.5:54321", "", "192.168.1.5")
	if got := clientIP(req); got != "203.0.113.5" {
		t.Fatalf("公网直连伪造 X-Real-IP 未被忽略: got=%q want=%q", got, "203.0.113.5")
	}
}

// 本机 nginx（对端 loopback）转发真实 LAN 客户端：信任 XFF 末值
func TestClientIP_TrustedProxyXFF(t *testing.T) {
	req := newReq("127.0.0.1:54321", "192.168.1.5", "")
	if got := clientIP(req); got != "192.168.1.5" {
		t.Fatalf("可信代理 XFF 未被信任: got=%q want=%q", got, "192.168.1.5")
	}
}

// 本机直连无转发头：返回 loopback 自身
func TestClientIP_LocalhostNoHeader(t *testing.T) {
	req := newReq("127.0.0.1:54321", "", "")
	if got := clientIP(req); got != "127.0.0.1" {
		t.Fatalf("本机直连异常: got=%q want=%q", got, "127.0.0.1")
	}
}

// 多段 XFF（客户端伪造前缀）经可信代理：取末值，丢弃前缀
func TestClientIP_XFFTakeLast(t *testing.T) {
	req := newReq("127.0.0.1:54321", "192.168.1.5, 203.0.113.9", "")
	if got := clientIP(req); got != "203.0.113.9" {
		t.Fatalf("多段 XFF 未取末值: got=%q want=%q", got, "203.0.113.9")
	}
}

// adminAllowed：非回环的真实 IPv6（含链路本地、文档前缀、ULA、公网 IPv6）一律拒绝
func TestAdminAllowed_IPv6Rejected(t *testing.T) {
	cases := []string{"fe80::1", "2001:db8::1", "fc00::1", "2408:8256:d287:3510:f814:a261:ac47:ded7"}
	for _, ip := range cases {
		if adminAllowed(clientInfo{IP: ip}, "") {
			t.Fatalf("非回环 IPv6 不应放行: %q", ip)
		}
	}
}

// adminAllowed：IPv4 映射 IPv6（::ffff:x.x.x.x）按映射到的 IPv4 地址判定公/私
func TestAdminAllowed_IPv4MappedIPv6(t *testing.T) {
	if adminAllowed(clientInfo{IP: "::ffff:8.8.8.8"}, "") {
		t.Fatalf("::ffff:8.8.8.8（公网 IPv4 映射）应拒绝")
	}
	if !adminAllowed(clientInfo{IP: "::ffff:192.168.1.5"}, "") {
		t.Fatalf("::ffff:192.168.1.5（私网 IPv4 映射）应放行")
	}
	if !adminAllowed(clientInfo{IP: "::ffff:127.0.0.1"}, "") {
		t.Fatalf("::ffff:127.0.0.1（回环映射）应放行")
	}
}

// adminAllowed：公网 IPv4 拒绝
func TestAdminAllowed_PublicV4Rejected(t *testing.T) {
	if adminAllowed(clientInfo{IP: "8.8.8.8"}, "") {
		t.Fatalf("公网 IPv4 不应放行")
	}
}

// adminAllowed：命中配置 CIDR 的 IPv4 放行
func TestAdminAllowed_CIDRHit(t *testing.T) {
	if !adminAllowed(clientInfo{IP: "192.168.1.50"}, "192.168.1.0/24") {
		t.Fatalf("命中 CIDR 的 IPv4 应放行")
	}
	if adminAllowed(clientInfo{IP: "192.168.2.50"}, "192.168.1.0/24") {
		t.Fatalf("未命中 CIDR 的 IPv4 不应放行")
	}
}

// adminAllowed：空 CIDR 时回退 IPv4 私网放行、公网拒绝
func TestAdminAllowed_EmptyCIDRFallback(t *testing.T) {
	if !adminAllowed(clientInfo{IP: "192.168.1.5"}, "") {
		t.Fatalf("空 CIDR 应回退 IPv4 私网放行")
	}
	if adminAllowed(clientInfo{IP: "8.8.8.8"}, "") {
		t.Fatalf("空 CIDR 公网 IPv4 不应放行")
	}
}

// adminAllowed：直连本机的 loopback 始终放行（即使不在 CIDR）
func TestAdminAllowed_LoopbackDirect(t *testing.T) {
	if !adminAllowed(clientInfo{IP: "127.0.0.1", Proxied: false}, "10.0.0.0/8") {
		t.Fatalf("直连 127.0.0.1 应始终放行")
	}
	if !adminAllowed(clientInfo{IP: "::1", Proxied: false}, "10.0.0.0/8") {
		t.Fatalf("直连 ::1 应始终放行")
	}
}

// adminAllowed：经代理转发来的 loopback 不再被当作“本机管理员”自动放行，
// 而是按真实 IP 继续判定（防止公网请求经代理被误判为本机）。
func TestAdminAllowed_LoopbackProxiedNotAuto(t *testing.T) {
	// 转发头声称是 127.0.0.1，但属于经代理转发：仍按私网回退放行（127.0.0.1 本身是私网），
	// 关键是它不会绕过下方的公网/IPv6 拒绝逻辑。此处验证公网经代理转发被拒。
	if adminAllowed(clientInfo{IP: "8.8.8.8", Proxied: true}, "") {
		t.Fatalf("经代理转发的公网 IP 不应放行")
	}
	// 经代理转发的公网 IPv6 必须拒绝
	if adminAllowed(clientInfo{IP: "2408:8256::1", Proxied: true}, "") {
		t.Fatalf("经代理转发的公网 IPv6 不应放行")
	}
}
