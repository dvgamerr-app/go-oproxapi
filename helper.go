package daas

import (
	"fmt"
	"math"
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gofiber/fiber/v2"
)

const (
	ENV        = "ENV"
	DEBUG      = "DEBUG"
	SENTRY_ENV = "SENTRY_ENV"
	SENTRY_DSN = "SENTRY_DSN"
)

type HTTP struct {
	Code  int
	Error error
}

type SubSet []string

func (s *SubSet) ToParam() string {
	return fmt.Sprintf("{%s}", strings.Join(*s, ","))
}
func (s *SubSet) Find(val string) int {
	for ix, v := range *s {
		if v == val {
			return ix
		}
	}
	return len(*s)
}

// Round math Round decimal
func Round(n float64, m float64) float64 {
	return math.Round(n*math.Pow(10, m)) / math.Pow(10, m)
}

func Estimated(start time.Time) int {
	duration, _ := elapsedDuration(start)
	return int(float64(duration.Microseconds()) / 1000)
}

func EstimatedPrint(start time.Time, name string, ctx ...*fiber.Ctx) {
	if os.Getenv(DEBUG) == "false" && os.Getenv(ENV) == "production" {
		return
	}
	_, elapsed := elapsedDuration(start)

	pc, _, _, _ := runtime.Caller(1)
	funcObj := runtime.FuncForPC(pc)
	if name == "" {
		runtimeFunc := regexp.MustCompile(`^.*\.(.*)$`)
		name = runtimeFunc.ReplaceAllString(funcObj.Name(), "$1")
	}
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// Debugf("%s # %s estimated. | alloc: %vMiB (%vMiB), sys: %vMiB, gc: %vMiB", name, elapsed, bToMb(m.Alloc), bToMb(m.TotalAlloc), bToMb(m.Sys), m.NumGC)

	if len(ctx) != 0 && ctx[0] != nil {
		ctx[0].Append("Server-Timing", fmt.Sprintf("app;dur=%v", elapsed))
	}
	Infof("%s # %s estimated.", name, elapsed)
}

func elapsedDuration(start time.Time) (time.Duration, string) {
	duration := time.Since(start)

	elapsed := ""
	if duration.Nanoseconds() < 1000 {
		elapsed = fmt.Sprintf("%dns", duration.Nanoseconds())
	} else if duration.Microseconds() < 1000 {
		elapsed = fmt.Sprintf("%0.3fμs", Round(float64(duration.Nanoseconds())/1000, 2))
	} else if duration.Milliseconds() < 1000 {
		elapsed = fmt.Sprintf("%0.3fms", Round(float64(duration.Microseconds())/1000, 2))
	} else if duration.Seconds() < 60 {
		elapsed = fmt.Sprintf("%0.3fms", Round(float64(duration.Microseconds())/1000, 2))
	} else {
		elapsed = fmt.Sprintf("%0.3fm", Round(float64(duration.Seconds()/60), 2))
	}
	return duration, elapsed
}

func TraceHandlerThrow(c *fiber.Ctx, code int, err error, ignoreCapture ...bool) error {
	Error(err)
	if len(ignoreCapture) > 0 {
		sentry.CaptureException(err)
	}
	return c.Status(code).JSON((HTTP{Code: code, Error: err}))
}

func TraceIsError(err error, stx *PGTx, skip ...bool) bool {
	if err != nil {
		Error(err)
		if len(skip) > 0 && skip[0] {
			sentry.CaptureException(err)
		}
		if stx != nil && !stx.Closed {
			stx.Rollback()
		}
	}
	return err != nil
}
