import (
	"fmt"
	"log"
)

func Printf(format string, args ...interface{}) {
msg := fmt.Sprintf(format, args...)
log.Printf("%s\n", msg)
}

func Fatal(format string, args ...interface{}) {
msg := fmt.Sprintf(format, args...)
log.Fatalf("%s\n", msg)
}
