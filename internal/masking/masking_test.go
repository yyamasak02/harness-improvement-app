package masking

import "testing"

func TestMask(t *testing.T) {
	m, err := New([]Extra{{Name: "proj", Regex: `PRJ-\d+`}})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct{ name, in, want string }{
		{"anthropic", "key sk-ant-api03-ABC_def-1 end", "key [MASKED:anthropic-key] end"},
		{"aws", "AKIADUMMYDUMMY012345", "[MASKED:aws-key]"},
		{"ghp", "ghp_abcdef123456", "[MASKED:github-token]"},
		{"github_pat", "github_pat_11AAAA_bbbbbb", "[MASKED:github-token]"},
		{"slack", "xoxb-12345-abcdef", "[MASKED:slack-token]"},
		{"bearer", "Authorization: Bearer abc.def-123", "Authorization: [MASKED:bearer]"},
		{"email", "mail a.b@example.co.jp now", "mail [MASKED:email] now"},
		{"kv colon", "password: hunter2", "password: [MASKED:secret]"},
		{"kv eq", "API_KEY=zzz", "API_KEY=[MASKED:secret]"},
		{"flag 空白区切り", "run --token abc123 now", "run --token [MASKED:secret] now"},
		{"flag 等号", "--api-key=abc123", "--api-key=[MASKED:secret]"},
		{"flag 他の語は対象外", "--tokens 5 --token-limit 3", "--tokens 5 --token-limit 3"},
		{"flag 二重引用符", `--token "abc def" ok`, `--token [MASKED:secret] ok`},
		{"flag 単引用符", `--password 'abc def' ok`, `--password [MASKED:secret] ok`},
		{"flag 等号と引用符", `--api-key="abc def" ok`, `--api-key=[MASKED:secret] ok`},
		{"語中のハイフンは対象外", "max-token 100 and multi-token sequences", "max-token 100 and multi-token sequences"},
		{"行頭のフラグ", "--secret abc123", "--secret [MASKED:secret]"},
		{"pem", "x\n-----BEGIN RSA PRIVATE KEY-----\nAAA\n-----END RSA PRIVATE KEY-----\ny", "x\n[MASKED:private-key]\ny"}, // privacy-allow: 検出テスト用のダミー
		{"pem 終端なし", "-----BEGIN PRIVATE KEY-----\nAAA", "[MASKED:private-key]"},                                           // privacy-allow: 検出テスト用のダミー
		{"custom", "see PRJ-42", "see [MASKED:proj]"},
		{"何も無し", "ふつうの文章", "ふつうの文章"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := m.Mask(tt.in); got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}

func TestInvalidExtra(t *testing.T) {
	if _, err := New([]Extra{{Name: "x", Regex: "("}}); err == nil {
		t.Fatal("エラーになるべき")
	}
}
