package offload

// 占位文本渲染面与解析面的对称契约（issue #284）。
//
// 解析面按空格切字段；渲染面对宿主自由文本（Kind/MimeType）原样写入，于是
// Kind="image large" 往返后变成 "image" 且 ok=true——字段被静默截断。修法是把
// 字段值里的空格等不安全字符转义，并让解析面遇到裸词直接判失败。

import (
	"strings"
	"testing"
)

// TestRenderParseRoundTripWithSpaces 含空格的自由文本字段必须原样往返。
func TestRenderParseRoundTripWithSpaces(t *testing.T) {
	cases := []Ref{
		{Kind: "image large", ID: "abcdef012345", Path: "blobs/ab/abcdef0123456789.png", Bytes: 10, MimeType: "image/png"},
		{Kind: "图像 大图", ID: "abcdef", Path: "blobs/ab/abcdef.png", Bytes: 1, MimeType: "text/plain; charset=utf-8"},
		{Kind: "a=b%c]d", ID: "abcdef", Path: "blobs/ab/abcdef.png", Bytes: 2, MimeType: "a[b] c=d"},
	}
	for _, want := range cases {
		got, ok := Parse(Render(want, ""))
		if !ok {
			t.Fatalf("含空格字段的占位文本应可解析: %+v", want)
		}
		if got != want {
			t.Fatalf("往返丢字段:\n got %+v\nwant %+v", got, want)
		}
	}
}

// TestParseRejectsBareWordInFields 字段区里的裸词是破损文本，不得静默跳过。
func TestParseRejectsBareWordInFields(t *testing.T) {
	cases := []string{
		"[外置对象 kind=image large id=abc path=blobs/ab/x bytes=1]",
		"[外置对象 kind=image id=abc path=blobs/ab/x bytes=1 裸词]",
		"[外置对象 kind=image  id=abc path=blobs/ab/x bytes=1]",
	}
	for _, in := range cases {
		if got, ok := Parse(in); ok {
			t.Fatalf("破损文本应判失败，实得 %+v: %s", got, in)
		}
	}
}

// TestRenderEscapesFieldSeparators 渲染值里不得出现裸空格（否则解析面按空格切碎）。
func TestRenderEscapesFieldSeparators(t *testing.T) {
	out := Render(Ref{Kind: "image large", ID: "abc", Path: "blobs/ab/abc.png", Bytes: 1}, "")
	field := out[strings.Index(out, markerPrefix)+len(markerPrefix):]
	field = field[:strings.Index(field, markerSuffix)]
	if strings.Contains(field, "kind=image large") {
		t.Fatalf("Kind 的空格必须转义: %s", out)
	}
	if strings.TrimSpace(field) == "" || strings.Contains(field, "  ") {
		t.Fatalf("字段区不应出现连续空白: %q", field)
	}
}
