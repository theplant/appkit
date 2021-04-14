package service

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/theplant/appkit/log"
)

func TestInstallErrorNotifier(t *testing.T) {
	ctx := context.Background()
	l := log.Default()

	t.Run("airbrake", func(t *testing.T) {
		os.Setenv("AIRBRAKE_PROJECTID", "1")
		os.Setenv("AIRBRAKE_TOKEN", "token")
		os.Setenv("AIRBRAKE_KEYSBLOCKLIST", "- Authorization\n")
		notifier, closer, _ := installErrorNotifier(ctx, l)
		defer closer.Close()
		typ := fmt.Sprintf("%T", notifier)
		if typ != "*errornotifier.airbrakeNotifier" {
			t.Fatalf("want notifier type is *errornotifier.airbrakeNotifier but get %s", typ)
		}
	})
}
