package setting

import (
	"fmt"
	"net"
	"strings"
	"sync"
)

// Codex 客户端限制 - 可信任请求 IP 白名单（命中则跳过客户端校验）
//
// 配置格式：
// - 支持单个 IP：1.2.3.4 / 2001:db8::1
// - 支持 CIDR：1.2.3.0/24 / 2001:db8::/32
// - 支持分隔符：换行、空格、逗号、分号、Tab
var CodexClientRestrictionTrustedIPWhitelist = ""

type codexClientRestrictionIPRule struct {
	ip    net.IP
	ipNet *net.IPNet
}

var codexClientRestrictionTrustedIPMutex sync.RWMutex
var codexClientRestrictionTrustedIPRules []codexClientRestrictionIPRule

func NormalizeCodexClientRestrictionTrustedIPWhitelist(value string) (string, []codexClientRestrictionIPRule, error) {
	tokens := strings.FieldsFunc(value, func(r rune) bool {
		switch r {
		case '\n', '\r', '\t', ' ', ',', ';':
			return true
		default:
			return false
		}
	})

	seen := make(map[string]struct{}, len(tokens))
	normalizedTokens := make([]string, 0, len(tokens))
	rules := make([]codexClientRestrictionIPRule, 0, len(tokens))

	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}

		// CIDR
		if strings.Contains(token, "/") {
			_, ipNet, err := net.ParseCIDR(token)
			if err != nil {
				return "", nil, fmt.Errorf("IP 白名单中存在非法 CIDR：%s", token)
			}
			canonical := ipNet.String()
			if _, ok := seen[canonical]; ok {
				continue
			}
			seen[canonical] = struct{}{}
			normalizedTokens = append(normalizedTokens, canonical)
			rules = append(rules, codexClientRestrictionIPRule{ipNet: ipNet})
			continue
		}

		// 单 IP
		ip := net.ParseIP(token)
		if ip == nil {
			return "", nil, fmt.Errorf("IP 白名单中存在非法 IP：%s", token)
		}
		canonical := ip.String()
		if _, ok := seen[canonical]; ok {
			continue
		}
		seen[canonical] = struct{}{}
		normalizedTokens = append(normalizedTokens, canonical)
		rules = append(rules, codexClientRestrictionIPRule{ip: ip})
	}

	return strings.Join(normalizedTokens, "\n"), rules, nil
}

func UpdateCodexClientRestrictionTrustedIPWhitelist(value string) error {
	normalized, rules, err := NormalizeCodexClientRestrictionTrustedIPWhitelist(value)
	if err != nil {
		return err
	}

	codexClientRestrictionTrustedIPMutex.Lock()
	CodexClientRestrictionTrustedIPWhitelist = normalized
	codexClientRestrictionTrustedIPRules = rules
	codexClientRestrictionTrustedIPMutex.Unlock()

	return nil
}

func GetCodexClientRestrictionTrustedIPWhitelist() string {
	codexClientRestrictionTrustedIPMutex.RLock()
	defer codexClientRestrictionTrustedIPMutex.RUnlock()
	return CodexClientRestrictionTrustedIPWhitelist
}

func IsCodexClientRestrictionTrustedIP(ipStr string) bool {
	ip := net.ParseIP(strings.TrimSpace(ipStr))
	if ip == nil {
		return false
	}

	codexClientRestrictionTrustedIPMutex.RLock()
	rules := codexClientRestrictionTrustedIPRules
	codexClientRestrictionTrustedIPMutex.RUnlock()

	for _, rule := range rules {
		if rule.ipNet != nil && rule.ipNet.Contains(ip) {
			return true
		}
		if rule.ip != nil && rule.ip.Equal(ip) {
			return true
		}
	}
	return false
}

