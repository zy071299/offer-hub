package tools

import "testing"

func TestFilterSensitiveWords(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"普通评论🙂", "普通评论🙂"}, {"", ""},
		{"傻逼，别说他妈的", "**，别说***"},
		{"fuck FucK", "**** ****"}, {"操你妈🙂", "***🙂"},
	} {
		if got := FilterSensitiveWords(tc.input); got != tc.want {
			t.Fatalf("filtered=%q want=%q", got, tc.want)
		}
	}
}
