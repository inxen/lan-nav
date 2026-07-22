package main

import (
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// clientInfo 描述解析出的客户端信息。
type clientInfo struct {
	IP      string // 客户端 IP：直连为 TCP 对端地址；经可信代理则为转发头中的真实客户端
	Proxied bool   // 该 IP 是否来自可信代理的转发头（而非直连对端）
}

// clientIPInfo 解析客户端 IP 与是否经代理转发。
//
// 安全模型：仅当 TCP 对端是可信代理（本机回环 / 本机其它网卡 IP / TRUSTED_PROXY 声明的 IP）
// 时，才信任其写入的 X-Real-IP / X-Forwarded-For；否则一律使用真实对端地址。
// 这样公网攻击者即使伪造 XFF / X-Real-IP 头，也无法冒充私网 IP 绕过后台白名单。
func clientIPInfo(r *http.Request) clientInfo {
	peer, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		peer = r.RemoteAddr
	}
	// 非可信代理（公网/局域网直连）：对端即真实客户端，不得使用其伪造的转发头
	if !isTrustedProxy(peer) {
		return clientInfo{IP: peer, Proxied: false}
	}
	// 仅本机回环 / 本机其它网卡 IP（如 nginx 经 192.168.1.10 等同机 IP 连接）等可信代理，
	// 才信任其写入的 X-Real-IP / X-Forwarded-For。
	if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
		return clientInfo{IP: xri, Proxied: true}
	}
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		// 取末值（最靠近真实客户端的那一段），丢弃客户端自行伪造的前缀
		parts := strings.Split(xff, ",")
		return clientInfo{IP: strings.TrimSpace(parts[len(parts)-1]), Proxied: true}
	}
	// 可信代理却未携带转发头：无法获知真实客户端 IP（通常是反向代理配置缺失）。
	// 记录一次警告，便于排查；此时回退为使用对端地址（本机回环）。
	warnNoForwardOnce.Do(func() {
		log.Printf("[警告] 可信代理(%s)连接但未携带 X-Real-IP / X-Forwarded-For，无法获取真实客户端IP；管理后台与场景判定将按本机回环处理。请检查反向代理配置（参考 deploy/）。", peer)
	})
	return clientInfo{IP: peer, Proxied: false}
}

// clientIP 兼容旧调用：仅返回客户端 IP 字符串。
func clientIP(r *http.Request) string { return clientIPInfo(r).IP }

var warnNoForwardOnce sync.Once

// isTrustedProxy 判断 TCP 对端是否为可信代理。
//
// 可信代理包括：
//   - 本机回环（127.0.0.1 / ::1）；
//   - 本机其它网卡 IP（nginx 与 lan-nav 同机，但经 192.168.1.10 等非回环 IP 连接）；
//   - 私网对端（如网关/反代所在的私网段，例：10.0.0.1）——它们转发真实客户端 IP，
//     应被信任；公网直连的对端是公网 IP，不会命中此处，故无法伪造转发头绕过白名单；
//   - 环境变量 TRUSTED_PROXY 显式声明的代理 IP/CIDR（用于反代部署在公网/其它网段的场景，
//     例如 Cloudflare 等 CDN 的边缘节点）。
//
// 注意：若对端不在上述之列（例如公网/局域网直连且非私网），一律视为真实客户端，
// 忽略其伪造的 X-Real-IP / X-Forwarded-For 头。
func isTrustedProxy(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	if ip.IsLoopback() {
		return true
	}
	// 本机其它网卡 IP：nginx 与 lan-nav 同机但经非回环 IP 连接时，对端仍是本机
	if localIPs[ip.String()] {
		return true
	}
	// 私网对端默认视为可信代理（网关/反代通常位于私网段，如 10.0.0.1）。
	// 公网直连的对端是公网 IP，不会被此处命中，因此无法冒充私网客户端绕过白名单。
	if ip.IsPrivate() {
		return true
	}
	for _, n := range trustedProxyNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// localIPs 缓存本机所有网卡 IP，用于识别“nginx 与 lan-nav 同机、但经非回环 IP 连接”的可信代理。
var localIPs = func() map[string]bool {
	m := map[string]bool{}
	if addrs, err := net.InterfaceAddrs(); err == nil {
		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok {
				m[ipnet.IP.String()] = true
			}
		}
	}
	return m
}()

// trustedProxyNets 来自环境变量 TRUSTED_PROXY（逗号分隔 IP/CIDR），
// 用于 nginx 部署在其它机器、需要显式声明其 IP 的场景。
var trustedProxyNets = func() []net.IPNet {
	var nets []net.IPNet
	for _, raw := range strings.Split(strings.TrimSpace(os.Getenv("TRUSTED_PROXY")), ",") {
		cidr := strings.TrimSpace(raw)
		if cidr == "" {
			continue
		}
		if ip := net.ParseIP(cidr); ip != nil {
			bits := 32
			if ip.To4() == nil {
				bits = 128
			}
			nets = append(nets, net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)})
			continue
		}
		if _, n, err := net.ParseCIDR(cidr); err == nil {
			nets = append(nets, *n)
		}
	}
	return nets
}()

// isPrivateIP 判断是否为局域网私有地址
func isPrivateIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	if ip.IsLoopback() {
		return true
	}
	if ip.IsPrivate() {
		return true
	}
	return false
}

// isInLAN 按配置网段判定是否为局域网
// lanCIDRs 为空时回退到私有地址判定；单个 CIDR 解析失败则跳过
func isInLAN(ipStr, lanCIDRs string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	if lanCIDRs == "" {
		return isPrivateIP(ipStr)
	}
	for _, raw := range strings.Split(lanCIDRs, ",") {
		cidr := strings.TrimSpace(raw)
		if cidr == "" {
			continue
		}
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// adminAllowed 判断是否允许访问后台管理。
//
// 规则：
//   - loopback（127.0.0.1 / ::1）始终放行（管理员在服务器本机访问自己的后台）；
//   - 非 loopback 的 IPv6 地址一律拒绝；
//   - 用户配置了 lan_cidrs → 在 CIDR 范围内才放行；
//   - lan_cidrs 留空 → 回退到仅允许 IPv4 私网（防新部署锁死入口），公网 IPv4 拒绝。
//
// 安全边界清晰：非 loopback IPv6（含公网 IPv6、IPv4 映射 IPv6 如 ::ffff:x.x.x.x
// 映射到公网 IP）在任何情况下都不会通过。
func adminAllowed(ci clientInfo, lanCIDRs string) bool {
	ip := net.ParseIP(ci.IP)
	if ip == nil {
		return false
	}

	// 仅「直连本机」(非经代理转发) 的回环视为本地管理员。
	// 经代理转发时，对端是网关/nginx，必须用真实客户端 IP 判定，
	// 不能把代理回环当本地（否则公网请求经代理也会变成“本机”而放行）。
	if ip.IsLoopback() && !ci.Proxied {
		return true
	}

	// 非 loopback 的任何 IPv6 形式一律拒绝（含公网 IPv6、::ffff: 映射）。
	// 用 To4() 而非字符串检测，避免与端口/方括号等边界问题耦合。
	if ip.To4() == nil {
		return false
	}

	// ---- 以下仅处理 IPv4 ----

	// 用户配置了 CIDR → 精确匹配
	if lanCIDRs != "" {
		return isInLAN(ci.IP, lanCIDRs)
	}

	// 未配置 CIDR → 仅允许 IPv4 私网（防锁死入口也防公网渗透）
	return ip.IsPrivate()
}

// ipWhitelist 管理接口 IP 白名单中间件
// 仅允许命中配置 CIDR 的 IPv4 私网访问，公网返回 404 隐藏入口。
// 判定经由 adminAllowed：任何 IPv6 一律拒绝，杜绝伪造私网 IP 绕过。
func ipWhitelist(store *Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		lanCIDRs := store.GetSetting("lan_cidrs", "")
		ci := clientIPInfo(c.Request)
		if !adminAllowed(ci, lanCIDRs) {
			log.Printf("[安全] 拒绝访问管理API: %s (路径=%s)", ci.IP, c.Request.URL.Path)
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		c.Next()
	}
}
