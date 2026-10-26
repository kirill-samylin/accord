package worker

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// setupPrompter is the one line-oriented dialogue of `worker setup`. Every
// answer comes from the owner's stdin; nothing is read from the network and
// no answer is ever echoed back as a secret (the dialogue asks for file paths,
// never for token contents).
type setupPrompter struct {
	in  *bufio.Reader
	out io.Writer
}

func newSetupPrompter(in io.Reader, out io.Writer) *setupPrompter {
	return &setupPrompter{in: bufio.NewReader(in), out: out}
}

func (p *setupPrompter) say(format string, args ...any) {
	fmt.Fprintf(p.out, format+"\n", args...)
}

// line reads one answer. EOF before any answer aborts the dialogue: the
// owner's absence never turns into a default consent.
func (p *setupPrompter) line() (string, error) {
	text, err := p.in.ReadString('\n')
	if err != nil && (err != io.EOF || text == "") {
		return "", errors.New("setup dialogue ended before the owner answered")
	}
	if len(text) > 4096 {
		return "", errors.New("setup answer is too long")
	}
	return strings.TrimSpace(text), nil
}

// ask prints a question with a default and returns the answer or the default.
func (p *setupPrompter) ask(question, fallback string) (string, error) {
	if fallback != "" {
		fmt.Fprintf(p.out, "%s [%s]: ", question, fallback)
	} else {
		fmt.Fprintf(p.out, "%s: ", question)
	}
	answer, err := p.line()
	if err != nil {
		return "", err
	}
	if answer == "" {
		return fallback, nil
	}
	return answer, nil
}

// confirm asks a yes/no question in Russian; empty input takes the default.
func (p *setupPrompter) confirm(question string, fallback bool) (bool, error) {
	hint := "д/Н"
	if fallback {
		hint = "Д/н"
	}
	for {
		fmt.Fprintf(p.out, "%s [%s]: ", question, hint)
		answer, err := p.line()
		if err != nil {
			return false, err
		}
		switch strings.ToLower(answer) {
		case "":
			return fallback, nil
		case "д", "да", "y", "yes":
			return true, nil
		case "н", "нет", "n", "no":
			return false, nil
		}
		p.say("Ответьте «да» или «нет».")
	}
}

// choose prints numbered options and returns the chosen index. The default
// index is taken on empty input; an out-of-range number is asked again.
func (p *setupPrompter) choose(question string, options []string, fallback int) (int, error) {
	for {
		p.say("%s", question)
		for i, option := range options {
			marker := " "
			if i == fallback {
				marker = "*"
			}
			p.say(" %s%d) %s", marker, i+1, option)
		}
		fmt.Fprintf(p.out, "Выбор [%d]: ", fallback+1)
		answer, err := p.line()
		if err != nil {
			return 0, err
		}
		if answer == "" {
			return fallback, nil
		}
		n, err := strconv.Atoi(answer)
		if err == nil && n >= 1 && n <= len(options) {
			return n - 1, nil
		}
		p.say("Введите номер от 1 до %d.", len(options))
	}
}

// splitList parses a comma- or space-separated answer into unique items.
func splitList(answer string) []string {
	var items []string
	seen := map[string]bool{}
	for _, item := range strings.FieldsFunc(answer, func(r rune) bool { return r == ',' || r == ' ' || r == ';' }) {
		if item != "" && !seen[item] {
			seen[item] = true
			items = append(items, item)
		}
	}
	return items
}
