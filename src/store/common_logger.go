package store

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// This log writer sends output to a file
type CommonLogWriter struct {
	mu  sync.Mutex
	rec chan string
	rot chan bool

	// The opened file
	filename string
	file     *os.File

	// Rotate at linecount
	maxlines          int
	maxlines_curlines int

	// Rotate at size
	maxsize         int
	maxsize_cursize int

	// Rotate daily
	daily          bool
	daily_opendate int

	// Keep old logfiles (.001, .002, etc)
	rotate bool
}

// This is the CommonLogWriter's output method
func (w *CommonLogWriter) Write(rec string) error {
	w.rec <- rec
	return nil
}

func (w *CommonLogWriter) Close() {
	close(w.rec)
	w.file.Sync()
}

// NewCommonLogWriter creates a new LogWriter which writes to the given file and
// has rotation enabled if rotate is true.
//
// If rotate is true, any time a new log file is opened, the old one is renamed
// with a .### extension to preserve it.  The various Set* methods can be used
// to configure log rotation based on lines, size, and daily.
//
// The standard log-line format is:
//   [%D %T] [%L] (%S) %M
func NewCommonLogWriter(fname string, rotate bool) *CommonLogWriter {
	w := &CommonLogWriter{
		rec:      make(chan string, 32),
		rot:      make(chan bool),
		filename: fname,
		rotate:   rotate,
	}

	// open the file for the first time
	if err := w.intRotate(); err != nil {
		fmt.Fprintf(os.Stderr, "CommonLogWriter(%q): %s\n", w.filename, err)
		return nil
	}

	go func() {
		defer func() {
			if w.file != nil {
				w.file.Close()
			}
		}()

		for {
			select {
			case <-w.rot:
				if err := w.intRotate(); err != nil {
					fmt.Fprintf(os.Stderr, "CommonLogWriter(%q): %s\n", w.filename, err)
					return
				}
			case rec, ok := <-w.rec:
				if !ok {
					return
				}
				now := time.Now()
				if (w.maxlines > 0 && w.maxlines_curlines >= w.maxlines) ||
					(w.maxsize > 0 && w.maxsize_cursize >= w.maxsize) ||
					(w.daily && now.Day() != w.daily_opendate) {
					if err := w.intRotate(); err != nil {
						fmt.Fprintf(os.Stderr, "CommonLogWriter(%q): %s\n", w.filename, err)
						return
					}
				}

				// Perform the write
				n, err := fmt.Fprint(w.file, rec)
				if err != nil {
					fmt.Fprintf(os.Stderr, "CommonLogWriter(%q): %s\n", w.filename, err)
					return
				}

				// Update the counts
				w.maxlines_curlines++
				w.maxsize_cursize += n
			}
		}
	}()

	return w
}

// Request that the logs rotate
func (w *CommonLogWriter) Rotate() {
	w.rot <- true
}

// If this is called in a threaded context, it MUST be synchronized
func (w *CommonLogWriter) intRotate() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	// Close any log file that may be open
	if w.file != nil {
		w.file.Close()
	}

	// If we are keeping log files, move it to the next available number
	if w.rotate {
		_, err := os.Lstat(w.filename)
		if err == nil { // file exists
			// Find the next available number
			num := 1
			fname := ""
			for ; err == nil && num <= 999; num++ {
				fname = w.filename + fmt.Sprintf(".%s.%03d", time.Now().Format("2006-01-02"), num)
				_, err = os.Lstat(fname)
			}
			// return error if the last file checked still existed
			if err == nil {
				return fmt.Errorf("Rotate: Cannot find free log number to rename %s\n", w.filename)
			}

			// Rename the file to its newfound home
			err = os.Rename(w.filename, fname)
			if err != nil {
				return fmt.Errorf("Rotate: %s\n", err)
			}
		}
	}

	// Open the log file
	fd, err := os.OpenFile(w.filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)
	if err != nil {
		return err
	}
	w.file = fd

	now := time.Now()

	// Set the daily open date to the current date
	w.daily_opendate = now.Day()

	// initialize rotation values
	w.maxlines_curlines = 0
	w.maxsize_cursize = 0

	return nil
}

// Set rotate at linecount (chainable). Must be called before the first log
// message is written.
func (w *CommonLogWriter) SetRotateLines(maxlines int) *CommonLogWriter {
	//fmt.Fprintf(os.Stderr, "CommonLogWriter.SetRotateLines: %v\n", maxlines)
	w.maxlines = maxlines
	return w
}

// Set rotate at size (chainable). Must be called before the first log message
// is written.
func (w *CommonLogWriter) SetRotateSize(maxsize int) *CommonLogWriter {
	//fmt.Fprintf(os.Stderr, "CommonLogWriter.SetRotateSize: %v\n", maxsize)
	w.maxsize = maxsize
	return w
}

// Set rotate daily (chainable). Must be called before the first log message is
// written.
func (w *CommonLogWriter) SetRotateDaily(daily bool) *CommonLogWriter {
	//fmt.Fprintf(os.Stderr, "CommonLogWriter.SetRotateDaily: %v\n", daily)
	w.daily = daily
	return w
}

// SetRotate changes whether or not the old logs are kept. (chainable) Must be
// called before the first log message is written.  If rotate is false, the
// files are overwritten; otherwise, they are rotated to another file before the
// new log is opened.
func (w *CommonLogWriter) SetRotate(rotate bool) *CommonLogWriter {
	//fmt.Fprintf(os.Stderr, "CommonLogWriter.SetRotate: %v\n", rotate)
	w.rotate = rotate
	return w
}
