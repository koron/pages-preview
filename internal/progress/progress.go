// Package progress provides simple progress bar.
package progress

import (
	"fmt"
	"io"
	"os"
	"strings"
)

type Progress struct {
	message string
	max     int
	value   int
	percent int
}

func New(message string, max int) *Progress {
	return &Progress{
		message: message,
		max:     max,
		percent: -1,
	}
}

func (p *Progress) Proceed(delta int) bool {
	p.value += delta
	if p.value > p.max {
		p.value = p.max
	}
	percent := p.value * 100 / p.max
	if percent != p.percent {
		p.percent = percent
		p.update()
	}
	return p.value >= p.max
}

func (p *Progress) writer() io.Writer {
	return os.Stderr
}

func (p *Progress) update() {
	count := min(max(p.percent, 0), 100) / 2
	fmt.Fprint(p.writer(),
		"\r",
		fmt.Sprintf("%s %3d%% ", p.message, p.percent),
		"|",
		strings.Repeat("=", max(count, 0)),
		">",
		strings.Repeat(" ", 50-count),
		"|")
	if p.percent >= 100 {
		fmt.Println()
	}
}
