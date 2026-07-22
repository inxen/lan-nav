package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

// blockedNets 列出禁止服务端代发请求的目标地址范围（SSRF 防护）。
// 覆盖 IPv4 私网/回环/链路本地/CGNAT/保留段，以及 IPv6 回环/唯一本地/链路本地/组播。
// 说明：IPv4 映射 IPv6（::ffff:a.b.c.d）会被下面的 IPv4 段经 To4 命中拦截。
var blockedNets = parseBlockedCIDRs()

func parseBlockedCIDRs() []*net.IPNet {
	list := []string{
		"0.0.0.0/8", "10.0.0.0/8", "100.64.0.0/10", "127.0.0.0/8",
		"169.254.0.0/16", "172.16.0.0/12", "192.0.0.0/24", "192.0.2.0/24",
		"192.168.0.0/16", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24",
		"224.0.0.0/4", "240.0.0.0/4",
		"::1/128", "::/128", "fc00::/7", "fe80::/10", "ff00::/8",
	}
	nets := make([]*net.IPNet, 0, len(list))
	for _, s := range list {
		_, n, err := net.ParseCIDR(s)
		if err != nil {
			panic("bad blocked CIDR: " + s)
		}
		nets = append(nets, n)
	}
	return nets
}

// isBlockedIP 判断某个 IP 是否属于禁止访问的地址范围。
func isBlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	for _, n := range blockedNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// hostAllowed 校验主机（域名或 IP 字面量）。解析后的所有地址均须非私有/回环/
// 链路本地等；若任一地址被拦截则整体拒绝（同时抵御 DNS 重绑定）。
func hostAllowed(host string) error {
	if ip := net.ParseIP(host); ip != nil {
		if isBlockedIP(ip) {
			return fmt.Errorf("目标地址被禁止: %s", host)
		}
		return nil
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("无法解析主机 %s: %w", host, err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("主机 %s 无可用 IP", host)
	}
	for _, ip := range ips {
		if isBlockedIP(ip) {
			return fmt.Errorf("目标地址被禁止: %s", host)
		}
	}
	return nil
}

// newHTTPClient 创建 HTTP 客户端。
//   - blockInternal=true：拒绝私有/内网地址（保留给需要公开出网的代理场景，防止被当作 SSRF 跳板）；
//   - blockInternal=false：允许内网地址（用于管理后台抓取局域网图标，管理员已受 IP 白名单保护）。
//
// 两种情况都会校验重定向协议（禁止 ftp:// file:// 等），并在解析域名后校验整组地址（防 DNS 重绑定）。
func newHTTPClient(timeout time.Duration, blockInternal bool) *http.Client {
	tr := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			// 已是 IP 字面量：按需校验
			if ip := net.ParseIP(host); ip != nil {
				if blockInternal && isBlockedIP(ip) {
					return nil, fmt.Errorf("拒绝连接被禁止地址 %s", host)
				}
				d := net.Dialer{Timeout: 5 * time.Second}
				return d.DialContext(ctx, network, addr)
			}
			// 域名：解析并校验整组地址（任一带私有段即拒绝，防 DNS 重绑定）
			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, err
			}
			if len(ips) == 0 {
				return nil, fmt.Errorf("无法解析 %s", host)
			}
			if blockInternal {
				for _, ipa := range ips {
					if isBlockedIP(ipa.IP) {
						return nil, fmt.Errorf("拒绝连接被禁止地址 %s", host)
					}
				}
			}
			// 校验通过，仍按原域名连接（保证 TLS SNI / Host 头正确）
			d := net.Dialer{Timeout: 5 * time.Second}
			return d.DialContext(ctx, network, addr)
		},
		MaxIdleConns:        20,
		IdleConnTimeout:     30 * time.Second,
		TLSHandshakeTimeout: 5 * time.Second,
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: tr,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("重定向次数过多")
			}
			u := req.URL
			if u.Scheme != "http" && u.Scheme != "https" {
				return fmt.Errorf("不支持的重定向协议: %s", u.Scheme)
			}
			if blockInternal {
				if err := hostAllowed(u.Hostname()); err != nil {
					return err
				}
			}
			return nil
		},
	}
}
