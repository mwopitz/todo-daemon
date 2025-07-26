package util

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/mwopitz/todo-daemon/api/todopb"
)

var errFullDisk = errors.New("write: no space left on device")

type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) {
	return f(p)
}

func TestPrintTasks(t *testing.T) {
	buf := &bytes.Buffer{}
	now := timestamppb.Now()
	tasks := []*todopb.Task{
		{
			Id:          "1",
			Summary:     "foo",
			CreatedAt:   timestamppb.New(now.AsTime().Add(-2 * time.Hour)),
			CompletedAt: timestamppb.New(now.AsTime().Add(-1 * time.Hour)),
		},
		{
			Id:          "2",
			Summary:     "bar",
			CreatedAt:   timestamppb.New(now.AsTime().Add(-1 * time.Hour)),
			CompletedAt: now,
		},
		{
			Id:        "3",
			Summary:   "baz",
			CreatedAt: now,
		},
	}
	want := "#1 [✓] foo\n#2 [✓] bar\n#3 [ ] baz\n"
	if err := PrintTasks(buf, tasks); err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); got != want {
		t.Errorf("got: %q; want: %q", got, want)
	}
}

func TestPrintTasksToFullDisk(t *testing.T) {
	fullDisk := writerFunc(func(_ []byte) (int, error) {
		return 0, errFullDisk
	})
	tasks := []*todopb.Task{
		{
			Id:      "1",
			Summary: "test",
		},
	}
	want := errFullDisk
	if got := PrintTasks(fullDisk, tasks); !errors.Is(got, want) {
		t.Errorf("got: %v; want: %v", got, want)
	}
}
