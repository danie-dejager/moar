//go:build windows
// +build windows

package twin

import (
	"fmt"
	"io"
	"os"
	"sync/atomic"
	"syscall"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/windows"
	"golang.org/x/term"
)

// NOTE: Karma points for replacing TestInterruptableReader_blockedOnRead() with
// TestInterruptableReader_blockedOnReadImmediate() and fixing the Windows
// implementation here so that the tests pass.

type interruptableReaderImpl struct {
	base              *os.File
	shutdownRequested atomic.Bool
}

// NOTE: To work properly, this Read() should return immediately after somebody
// calls Interrupt(), *without first reading any bytes from the base reader*.
//
// This implementation doesn't do that. If you want to fix this, the not-Windows
// implementation in screen-setup.go may or may not work as inspiration.
func (r *interruptableReaderImpl) Read(p []byte) (n int, err error) {
	if r.shutdownRequested.Load() {
		err = io.EOF
		return
	}

	n, err = r.base.Read(p)
	if err != nil {
		return
	}

	if r.shutdownRequested.Load() {
		err = io.EOF
		n = 0
	}
	return
}

func (r *interruptableReaderImpl) Interrupt() {
	// Previously we used to close the screen.ttyIn file descriptor here, but:
	// * That didn't interrupt the blocking read() in the main loop
	// * It may or may not have caused shutdown issues on Windows
	//
	// Setting this flag doesn't interrupt the blocking read() either, but it
	// should at least not cause any shutdown issues on Windows.
	//
	// Ref:
	// * https://github.com/walles/moar/issues/217
	// * https://github.com/walles/moar/issues/221
	r.shutdownRequested.Store(true)
}

func newInterruptableReader(base *os.File) (interruptableReader, error) {
	return &interruptableReaderImpl{base: base}, nil
}

func (screen *UnixScreen) setupSigwinchNotification() {
	screen.sigwinch = make(chan int, 1)
	screen.sigwinch <- 0 // Trigger initial screen size query

	// No SIGWINCH handling on Windows for now, contributions welcome, see
	// sigwinch.go for inspiration.
}

func (screen *UnixScreen) setupTtyInTtyOut() error {
	in, err := syscall.Open("CONIN$", syscall.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("failed to open CONIN$: %w", err)
	}

	screen.ttyIn = os.NewFile(uintptr(in), "/dev/tty")

	// Set input stream to raw mode
	stdin := windows.Handle(screen.ttyIn.Fd())
	err = windows.GetConsoleMode(stdin, &screen.oldTtyInMode)
	if err != nil {
		return fmt.Errorf("failed to get stdin console mode: %w", err)
	}
	err = windows.SetConsoleMode(stdin, screen.oldTtyInMode|windows.ENABLE_VIRTUAL_TERMINAL_INPUT)
	if err != nil {
		return fmt.Errorf("failed to set stdin console mode: %w", err)
	}

	screen.oldTerminalState, err = term.MakeRaw(int(screen.ttyIn.Fd()))
	if err != nil {
		screen.restoreTtyInTtyOut() // Error intentionally ignored, report the first one only
		return fmt.Errorf("failed to set raw mode: %w", err)
	}

	screen.ttyOut = os.Stdout

	// Enable console colors, from: https://stackoverflow.com/a/52579002
	stdout := windows.Handle(screen.ttyOut.Fd())
	err = windows.GetConsoleMode(stdout, &screen.oldTtyOutMode)
	if err != nil {
		screen.restoreTtyInTtyOut() // Error intentionally ignored, report the first one only
		return fmt.Errorf("failed to get stdout console mode: %w", err)
	}
	err = windows.SetConsoleMode(stdout, screen.oldTtyOutMode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
	if err != nil {
		screen.restoreTtyInTtyOut() // Error intentionally ignored, report the first one only
		return fmt.Errorf("failed to set stdout console mode: %w", err)
	}

	ttyInTerminalState, err := term.GetState(int(screen.ttyIn.Fd()))
	if err != nil {
		return err
	}
	log.Info("ttyin terminal state: ", fmt.Sprintf("%+v", ttyInTerminalState))

	ttyOutTerminalState, err := term.GetState(int(screen.ttyOut.Fd()))
	if err != nil {
		return err
	}
	log.Info("ttyout terminal state: ", fmt.Sprintf("%+v", ttyOutTerminalState))

	return nil
}

func (screen *UnixScreen) restoreTtyInTtyOut() error {
	errors := []error{}

	stdin := windows.Handle(screen.ttyIn.Fd())
	err := windows.SetConsoleMode(stdin, screen.oldTtyInMode)
	if err != nil {
		errors = append(errors, fmt.Errorf("failed to restore stdin console mode: %w", err))
	}

	stdout := windows.Handle(screen.ttyOut.Fd())
	err = windows.SetConsoleMode(stdout, screen.oldTtyOutMode)
	if err != nil {
		errors = append(errors, fmt.Errorf("failed to restore stdout console mode: %w", err))
	}

	if len(errors) == 0 {
		return nil
	}

	return fmt.Errorf("failed to restore terminal state: %v", errors)
}
