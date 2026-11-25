package offload

// 扩展名与逻辑键判据的一致性（issue #283 的连带面）。
//
// 写入侧产出的键必须一定能被解析面接受。扩展名里的连续点会产出 `x..png` 这类
// 被 validRelPath 拒绝的键——若写入侧放过它，就出现「本包自己写的占位文本
// 自己解析不了」。ext 侧必须与键形状判据同口径拒绝。

import (
	"testing"

	"github.com/oekit/pkg/errors"
)

func TestSanitizeExtRejectsConsecutiveDots(t *testing.T) {
	for _, ext := range []string{"..png", "a..b", ".a..b"} {
		if _, err := StorePath("abcdef", ext); !errors.Is(err, ErrBadExt) {
			t.Fatalf("扩展名 %q 应返回 ErrBadExt，实得 %v", ext, err)
		}
	}
}
