package trace

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestTrace(t *testing.T) {
	ctx := context.Background()
	ctx, span := StartSpan(ctx, "top-level")
	defer func() { EndSpan(ctx, span, nil) }()

	ctx2, span2 := StartSpan(ctx, "second-level")
	span2.AddAttributes(
		StringAttribute("second-level-only", "test"),
	)
	span2.AddInheritableAttributes(
		StringAttribute("second-level-inheritable", "test"),
		StringAttribute("second-level-inheritable-shoul-be-override", "test"),
	)
	time.Sleep(2 * time.Second)
	defer func() { EndSpan(ctx2, span2, nil) }()

	ctx3, span3 := StartSpan(ctx2, "third-level")
	span3.AddAttributes(
		StringAttribute("second-level-inheritable-shoul-be-override", "override"),
	)
	time.Sleep(3 * time.Second)
	defer func() { EndSpan(ctx3, span3, errors.New("third-level-failed")) }()
}
