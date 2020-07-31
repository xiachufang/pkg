package logger

import "testing"

func TestLogger(t *testing.T) {
	logger, err := New(&Options{
		Tag:   "foo.bar",
		Debug: true,
	})

	if err != nil {
		t.Fatal("init logger error: ", err)
	}

	logger.Info("foo")
	logger.Debug("foo")
	logger.Error("foo")
}
