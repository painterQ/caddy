package painter_sign

import (
	"testing"

	"github.com/painterQ/poplar/logger"
)

func TestDing(t *testing.T) {
	l := logger.GetLogger("auth", logger.GetDingTalkOpt(dingToken))
	l.DingInfo("test", []string{"painterqiao"})
}
