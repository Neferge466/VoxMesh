package parser

import (
	"regexp"
	"strings"
)

// Command represents a parsed voice command.
type Command struct {
	Action string         `json:"action"`
	Params map[string]string `json:"params"`
}

var patterns = []struct {
	re     *regexp.Regexp
	action string
	params func(matches []string) map[string]string
}{
	{regexp.MustCompile(`(?i)(?:加入|切换到|去)\s*(.+?)\s*(?:频道)?\s*$`), "join_channel", func(m []string) map[string]string {
		return map[string]string{"channel_name": strings.TrimSpace(m[1])}
	}},
	{regexp.MustCompile(`(?i)(?:离开|退出)\s*(?:频道)?\s*$`), "leave_channel", nil},
	{regexp.MustCompile(`(?i)(?:静音|闭麦| mute$)`), "mute", nil},
	{regexp.MustCompile(`(?i)(?:开麦|取消静音| unmute$)`), "unmute", nil},
	{regexp.MustCompile(`(?i)(?:音量|声音)\s*(?:调|加大|增大|升高|调高)\s*$`), "volume_up", nil},
	{regexp.MustCompile(`(?i)(?:音量|声音)\s*(?:减小|降低|调低|调小)\s*$`), "volume_down", nil},
	{regexp.MustCompile(`(?i)(?:音量|声音)\s*(?:调到)?\s*(\d+)\s*$`), "volume_set", func(m []string) map[string]string {
		return map[string]string{"level": m[1]}
	}},
	{regexp.MustCompile(`(?i)(?:我在|当前).*(?:哪个|什么).*(?:频道)?\s*$`), "what_channel", nil},
	{regexp.MustCompile(`(?i)(?:谁|哪个).*(?:在说|说话|正在说)\s*$`), "who_speaking", nil},
	{regexp.MustCompile(`(?i)(?:打开|开启|启用)\s*(?:降噪|噪声抑制)\s*$`), "enable_nr", nil},
	{regexp.MustCompile(`(?i)(?:关闭|禁用|停用)\s*(?:降噪|噪声抑制)\s*$`), "disable_nr", nil},
}

// Parse extracts a command from transcript text.
func Parse(transcript string) *Command {
	text := strings.TrimSpace(transcript)
	for _, p := range patterns {
		matches := p.re.FindStringSubmatch(text)
		if matches != nil {
			cmd := &Command{Action: p.action}
			if p.params != nil {
				cmd.Params = p.params(matches)
			}
			return cmd
		}
	}
	return nil // unrecognized
}
