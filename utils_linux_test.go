package elasticapm

import (
	"syscall"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

func TestCurrentProcessTitle(t *testing.T) {
	origProcessTitle, err := currentProcessTitle()
	assert.NoError(t, err)

	setProcessTitle := func(title string) {
		var buf [16]byte
		copy(buf[:], title)
		if _, _, errno := syscall.RawSyscall6(
			syscall.SYS_PRCTL, syscall.PR_SET_NAME,
			uintptr(unsafe.Pointer(&buf[0])),
			0, 0, 0, 0,
		); errno != 0 {
			t.Fatal(errno)
		}
	}

	const desiredTitle = "foo [bar]"
	setProcessTitle(desiredTitle)
	defer setProcessTitle(origProcessTitle)

	processTitle, err := currentProcessTitle()
	assert.NoError(t, err)
	assert.Equal(t, desiredTitle, processTitle)
}
