package worker

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"strconv"
	"strings"
)

type testCounts struct{ Passed, Failed, Skipped int64 }

func parseTestReport(format string, raw []byte) (testCounts, error) {
	if len(raw) == 0 || len(raw) > 4<<20 {
		return testCounts{}, errors.New("bounded test report required")
	}
	switch format {
	case "junit":
		return parseJUnit(raw)
	case "tap":
		return parseTAP(raw)
	}
	return testCounts{}, errors.New("unsupported real test report")
}
func parseJUnit(raw []byte) (testCounts, error) {
	var out testCounts
	decoder := xml.NewDecoder(bytes.NewReader(raw))
	depth, cases := 0, 0
	root := ""
	declaredTests := -1
	declaredFailure := false
	inside := false
	failed, skipped := false, false
	for {
		token, e := decoder.Token()
		if e == io.EOF {
			break
		}
		if e != nil {
			return out, errors.New("invalid JUnit report")
		}
		switch v := token.(type) {
		case xml.StartElement:
			if depth == 0 && root != "" {
				return out, errors.New("multiple JUnit roots")
			}
			depth++
			if depth > 64 {
				return out, errors.New("JUnit depth exceeded")
			}
			if root == "" {
				root = v.Name.Local
				if root != "testsuite" && root != "testsuites" {
					return out, errors.New("JUnit suite required")
				}
			}
			if v.Name.Local == "testsuite" || v.Name.Local == "testsuites" {
				for _, attr := range v.Attr {
					if attr.Name.Local == "tests" || attr.Name.Local == "failures" || attr.Name.Local == "errors" {
						n, parseErr := strconv.Atoi(attr.Value)
						if parseErr != nil || n < 0 || n > 10000000 {
							return out, errors.New("invalid JUnit declared counts")
						}
						if depth == 1 && attr.Name.Local == "tests" {
							declaredTests = n
						}
						if (attr.Name.Local == "failures" || attr.Name.Local == "errors") && n > 0 {
							declaredFailure = true
						}
					}
				}
			}
			switch v.Name.Local {
			case "testcase":
				if inside {
					return out, errors.New("nested JUnit testcase")
				}
				inside = true
				failed, skipped = false, false
				cases++
				if cases > 1000000 {
					return out, errors.New("JUnit test count exceeded")
				}
			case "failure", "error":
				if !inside {
					return out, errors.New("JUnit suite failure")
				}
				failed = true
			case "skipped":
				if !inside {
					return out, errors.New("invalid skipped testcase")
				}
				skipped = true
			}
		case xml.EndElement:
			if v.Name.Local == "testcase" {
				if !inside {
					return out, errors.New("invalid JUnit testcase")
				}
				if failed {
					out.Failed++
				} else if skipped {
					out.Skipped++
				} else {
					out.Passed++
				}
				inside = false
			}
			depth--
		case xml.CharData:
			if depth == 0 && strings.TrimSpace(string(v)) != "" {
				return out, errors.New("JUnit data outside suite")
			}
		case xml.Directive:
			return out, errors.New("JUnit directives are unsupported")
		}
	}
	if root == "" || depth != 0 || inside || out.Passed+out.Failed < 1 || (declaredTests >= 0 && int64(declaredTests) != out.Passed+out.Failed+out.Skipped) || (declaredFailure && out.Failed == 0) {
		return out, errors.New("JUnit contains no executed tests")
	}
	return out, nil
}
func parseTAP(raw []byte) (testCounts, error) {
	var out testCounts
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 4096), 64<<10)
	plan := -1
	seen := map[int]bool{}
	tests := 0
	yaml := false
	for scanner.Scan() {
		rawLine := scanner.Text()
		line := strings.TrimSpace(rawLine)
		indented := len(rawLine) > 0 && (rawLine[0] == ' ' || rawLine[0] == '\t')
		if yaml {
			if line == "..." {
				yaml = false
			}
			continue
		}
		if indented {
			if line == "---" {
				yaml = true
			}
			continue
		}
		if line == "" || strings.HasPrefix(line, "#") || line == "TAP version 13" {
			continue
		}
		if strings.HasPrefix(line, "Bail out!") {
			return out, errors.New("TAP bailed out")
		}
		if strings.HasPrefix(line, "1..") {
			fields := strings.Fields(line)
			n, e := strconv.Atoi(strings.TrimPrefix(fields[0], "1.."))
			if e != nil || n < 1 || n > 1000000 || plan != -1 {
				return out, errors.New("invalid TAP plan")
			}
			plan = n
			continue
		}
		failed := strings.HasPrefix(line, "not ok ") || line == "not ok"
		ok := strings.HasPrefix(line, "ok ") || line == "ok"
		if !failed && !ok {
			return out, errors.New("unsupported TAP framing")
		}
		fields := strings.Fields(line)
		offset := 1
		if failed {
			offset = 2
		}
		number := tests + 1
		if len(fields) > offset {
			if n, e := strconv.Atoi(fields[offset]); e == nil {
				number = n
			}
		}
		if number != tests+1 || seen[number] {
			return out, errors.New("TAP sequence mismatch")
		}
		seen[number] = true
		tests++
		if tests > 1000000 {
			return out, errors.New("TAP count exceeded")
		}
		directive := ""
		if _, tail, found := strings.Cut(line, "#"); found {
			directive = strings.ToUpper(strings.TrimSpace(tail))
		}
		if strings.HasPrefix(directive, "SKIP") {
			out.Skipped++
		} else if strings.HasPrefix(directive, "TODO") {
			return out, errors.New("TAP TODO is not a verification pass")
		} else if failed {
			out.Failed++
		} else {
			out.Passed++
		}
	}
	if scanner.Err() != nil || yaml || plan != tests || out.Passed+out.Failed < 1 {
		return out, errors.New("TAP requires a complete nonempty executed plan")
	}
	return out, nil
}
