package utils

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// 云元数据服务常见主机名，一律拒绝
var metadataHosts = map[string]bool{
	"metadata":                     true,
	"metadata.google.internal":     true,
	"metadata.goog":                true,
	"instance-data":                true,
	"instance-data.ec2.internal":   true,
	"169.254.169.254":              true,
	"100.100.100.200":              true,
	"metadata.azure.internal":      true,
}

// ValidatePublicURL 校验服务端即将请求的外部 URL，阻断 SSRF。
//
// blockPrivate = true  ：用于用户可控 URL（图片链接、MJ 图片地址等），
//                        拒绝回环、链路本地、所有私有网段与云元数据地址。
// blockPrivate = false ：用于管理员自建渠道地址，仍拒绝回环与云元数据，
//                        但允许 192.168 / 10.x 等内网自建服务。
func ValidatePublicURL(raw string, blockPrivate bool) error {
	if strings.TrimSpace(raw) == "" {
		return errors.New("URL 为空")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return errors.New("URL 格式无效")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("仅支持 http/https 协议")
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return errors.New("URL 缺少主机名")
	}
	if metadataHosts[host] {
		return errors.New("目标地址不允许（云元数据服务）")
	}

	if ip := net.ParseIP(host); ip != nil {
		return validateIP(ip, blockPrivate)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil || len(addrs) == 0 {
		return errors.New("无法解析目标主机")
	}
	for _, addr := range addrs {
		if err := validateIP(addr.IP, blockPrivate); err != nil {
			return err
		}
	}
	return nil
}

func validateIP(ip net.IP, blockPrivate bool) error {
	if ip.IsLoopback() || ip.IsUnspecified() || ip.IsMulticast() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return errors.New("目标地址不允许（回环/链路本地/保留地址）")
	}
	if blockPrivate && ip.IsPrivate() {
		return errors.New("目标地址不允许（私有网段）")
	}
	return nil
}

// NewSafeHTTPClient 返回带超时和逐跳 SSRF 复查的 HTTP 客户端。
// 每一跳重定向都会重新校验，避免用公网 302 跳转到内网绕过检查。
func NewSafeHTTPClient(timeout time.Duration, blockPrivate bool) *http.Client {
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("重定向次数过多")
			}
			if err := ValidatePublicURL(req.URL.String(), blockPrivate); err != nil {
				return fmt.Errorf("重定向目标被拒绝: %w", err)
			}
			return nil
		},
	}
}

// SSRFCheckRedirect 可挂在已有自定义 Transport 的 http.Client 上。
func SSRFCheckRedirect(blockPrivate bool) func(*http.Request, []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return errors.New("重定向次数过多")
		}
		if err := ValidatePublicURL(req.URL.String(), blockPrivate); err != nil {
			return fmt.Errorf("重定向目标被拒绝: %w", err)
		}
		return nil
	}
}
