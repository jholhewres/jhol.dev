package uaclass

import "strings"

// CrawlerAgents lists known bot/crawler user-agent substrings (lowercase).
var CrawlerAgents = []string{
	"linkedinbot", "facebookexternalhit", "twitterbot",
	"slackbot", "telegrambot", "whatsapp", "googlebot",
	"bingbot", "yandexbot", "baiduspider", "duckduckbot",
}

// IsCrawler reports whether the given User-Agent string belongs to a known crawler.
func IsCrawler(ua string) bool {
	lower := strings.ToLower(ua)
	for _, bot := range CrawlerAgents {
		if strings.Contains(lower, bot) {
			return true
		}
	}
	return false
}

// ClassifyUA returns a simplified UA class: edge, chrome, firefox, safari, or other.
// Order matters: check Edge before Chrome because Edge UAs contain "Chrome".
func ClassifyUA(ua string) string {
	lower := strings.ToLower(ua)
	switch {
	case strings.Contains(lower, "edg"):
		return "edge"
	case strings.Contains(lower, "chrome"):
		return "chrome"
	case strings.Contains(lower, "firefox"):
		return "firefox"
	case strings.Contains(lower, "safari"):
		return "safari"
	default:
		return "other"
	}
}
