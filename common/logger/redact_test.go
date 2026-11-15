package logger

import (
	"strings"
	"testing"
)

func TestRedactSensitive(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "telegram bot token embedded in URL path",
			in:   `failed to execute POST request to getMe: Post "https://api.telegram.org/bot7569035391:AAHq8dummyTokenValue1234567890/getMe": dial tcp timeout`,
			want: `failed to execute POST request to getMe: Post "https://api.telegram.org/bot<redacted>/getMe": dial tcp timeout`,
		},
		{
			name: "feishu webhook",
			in:   `POST https://open.feishu.cn/open-apis/bot/v2/hook/8f3d7c2e-1111-4a2b-9c3d-abcdef123456 failed`,
			want: `POST https://open.feishu.cn/open-apis/bot/v2/hook/<redacted> failed`,
		},
		{
			name: "wecom webhook key",
			in:   `POST https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=693a91f6-7xxx-4bc4-97a0-0ec2sifa5aaa`,
			want: `POST https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=<redacted>`,
		},
		{
			name: "generic query credentials",
			in:   `GET /api/verify?access_token=abcdef123456&state=xyz987654`,
			want: `GET /api/verify?access_token=<redacted>&state=<redacted>`,
		},
		{
			name: "github PAT and sk key",
			in:   `auth ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 and sk-A6sz1HQ6DpsIjwn_fVAyxp8NBLI_ArhzUmkEmo1mI0c3c`,
			want: `auth <redacted> and <redacted>`,
		},
		{
			name: "ordinary log line unchanged",
			in:   `GIN request status=200 path=/v1/chat/completions model=gpt-4o-mini latency=1.2s`,
			want: `GIN request status=200 path=/v1/chat/completions model=gpt-4o-mini latency=1.2s`,
		},
		{
			name: "empty",
			in:   "",
			want: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := RedactSensitive(tc.in)
			if got != tc.want {
				t.Fatalf("got  %q\nwant %q", got, tc.want)
			}
			// 幂等：重复脱敏不应改变结果
			if again := RedactSensitive(got); again != got {
				t.Fatalf("not idempotent: %q -> %q", got, again)
			}
			if strings.Contains(got, "AAHq8dummy") || strings.Contains(got, "693a91f6") {
				t.Fatalf("credential still present: %q", got)
			}
		})
	}
}
