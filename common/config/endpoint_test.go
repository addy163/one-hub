package config

import "testing"

const testEndpoints = `[
  {"id":"main","name":"主线路","url":"https://api.example.com/","groups":["vip","svip"],"enabled":true,"sort":100},
  {"id":"free","name":"免费线路","url":"https://free.example.com","groups":["default"],"enabled":true,"sort":10},
  {"id":"all","name":"公共线路","url":"https://all.example.com","groups":[],"enabled":true,"sort":50},
  {"id":"off","name":"已停用","url":"https://off.example.com","enabled":false,"sort":999},
  {"id":"dup","name":"重复","url":"https://api.example.com","groups":["default"],"enabled":true,"sort":1},
  {"id":"bad","name":"非法","url":"ftp://bad.example.com","enabled":true,"sort":1}
]`

func TestParseRelayEndpoints(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		if got := ParseRelayEndpoints(""); len(got) != 0 {
			t.Fatalf("want empty, got %d", len(got))
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		if got := ParseRelayEndpoints("{not json"); len(got) != 0 {
			t.Fatalf("want empty, got %d", len(got))
		}
	})

	t.Run("drops illegal and duplicate, trims slash, sorts", func(t *testing.T) {
		got := ParseRelayEndpoints(testEndpoints)
		// Parse 只做清洗和排序（按 sort 降序），是否启用由 RelayEndpointsForGroup 过滤
		wantOrder := []string{"https://off.example.com", "https://api.example.com", "https://all.example.com", "https://free.example.com"}
		if len(got) != len(wantOrder) {
			t.Fatalf("want %d items, got %d: %+v", len(wantOrder), len(got), got)
		}
		for i, want := range wantOrder {
			if got[i].Url != want {
				t.Errorf("index %d: want %s, got %s", i, want, got[i].Url)
			}
		}
		for _, item := range got {
			if item.Url == "https://api.example.com" && item.Id != "main" {
				t.Errorf("duplicate url should keep the first one, got id %s", item.Id)
			}
		}
	})

	t.Run("fills id and name when missing", func(t *testing.T) {
		got := ParseRelayEndpoints(`[{"url":"https://x.example.com","enabled":true}]`)
		if len(got) != 1 {
			t.Fatalf("want 1 item, got %d", len(got))
		}
		if got[0].Id != "https://x.example.com" || got[0].Name != "https://x.example.com" {
			t.Errorf("unexpected fallback id/name: %+v", got[0])
		}
	})
}

func TestRelayEndpointsForGroup(t *testing.T) {
	cases := []struct {
		name  string
		group string
		want  []string
	}{
		{
			name:  "vip user sees vip, public, and not free-only",
			group: "vip",
			want:  []string{"https://api.example.com", "https://all.example.com"},
		},
		{
			name:  "default user sees free line and public line only",
			group: "default",
			want:  []string{"https://all.example.com", "https://free.example.com"},
		},
		{
			name:  "anonymous falls back to default",
			group: "",
			want:  []string{"https://all.example.com", "https://free.example.com"},
		},
		{
			name:  "unknown group only sees public lines",
			group: "nobody",
			want:  []string{"https://all.example.com"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := RelayEndpointsForGroup(testEndpoints, tc.group)
			if len(got) != len(tc.want) {
				t.Fatalf("want %d items, got %d: %+v", len(tc.want), len(got), got)
			}
			for i, want := range tc.want {
				if got[i].Url != want {
					t.Errorf("index %d: want %s, got %s", i, want, got[i].Url)
				}
				if got[i].Groups != nil {
					t.Errorf("groups must not be exposed, got %v", got[i].Groups)
				}
			}
		})
	}
}

func TestRelayEndpointsForGroupSkipsDisabled(t *testing.T) {
	for _, item := range RelayEndpointsForGroup(testEndpoints, "svip") {
		if item.Url == "https://off.example.com" {
			t.Fatal("disabled endpoint must never be delivered")
		}
	}
}
