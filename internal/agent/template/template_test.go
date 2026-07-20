package template

import "testing"

func TestMatchPrefersRollbackOverRelease(t *testing.T) {
	matched, ok := Match("回滚上次发布")
	if !ok || matched.ID != "rollback" {
		t.Fatalf("Match() = %+v, %v; want rollback", matched, ok)
	}
}

func TestMatchReturnsFalseForUnknownGoal(t *testing.T) {
	if matched, ok := Match("查看服务器信息"); ok {
		t.Fatalf("Match() = %+v, true; want false", matched)
	}
}
