package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type bufferCloser struct{ bytes.Buffer }

func (*bufferCloser) Close() error { return nil }

type rpcMockProcess struct {
	reader    *io.PipeReader
	writer    *io.PipeWriter
	done      chan struct{}
	once      sync.Once
	uncertain bool
}

func (p *rpcMockProcess) Output() io.ReadCloser { return p.reader }
func (p *rpcMockProcess) Wait() error           { <-p.done; return nil }
func (p *rpcMockProcess) Stop() error {
	p.once.Do(func() { p.writer.Close(); close(p.done) })
	if p.uncertain {
		return errOwnership
	}
	return nil
}
func (*rpcMockProcess) Forced() bool          { return false }
func (*rpcMockProcess) ExitCode() (int, bool) { return 0, true }

func TestConfigInspectionDialogueIsInteractiveFixedAndNeverPublishesRawConfig(t *testing.T) {
	i, config := permissionFixture(t)
	config["config"].(map[string]any)["api_key"] = "synthetic-secret-not-for-output"
	var methods []string
	start := func(spec launchSpec) (process, error) {
		r, w := io.Pipe()
		p := &rpcMockProcess{reader: r, writer: w, done: make(chan struct{})}
		go func() {
			defer p.Stop()
			decoder := json.NewDecoder(spec.stdin)
			encoder := json.NewEncoder(w)
			for _, method := range []string{"initialize", "initialized", "config/read", "configRequirements/read"} {
				var req struct {
					ID     int            `json:"id"`
					Method string         `json:"method"`
					Params map[string]any `json:"params"`
				}
				if decoder.Decode(&req) != nil || req.Method != method {
					return
				}
				methods = append(methods, method)
				if method == "initialized" {
					continue
				}
				var result any = map[string]any{"requirements": nil}
				if method == "initialize" {
					result = map[string]any{"userAgent": "synthetic", "platformFamily": "unix", "platformOs": "macos", "codexHome": "/private/synthetic"}
				}
				if method == "config/read" {
					if req.Params["cwd"] != i.Directory || req.Params["includeLayers"] != true {
						return
					}
					result = config
				}
				if encoder.Encode(map[string]any{"id": req.ID, "result": result}) != nil {
					return
				}
			}
			if decoder.Decode(new(any)) != io.EOF {
				return
			}
		}()
		return p, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	proof, _, err := inspectWithProcess(ctx, launchSpec{directory: i.Directory, inspection: &i}, start)
	if err != nil || !validSHA256(proof.Digest) || len(methods) != 4 {
		t.Fatal("fixed dialogue failed", err)
	}
	if strings.Contains(string(permissionRaw(proof)), "secret") {
		t.Fatal("raw config escaped safe proof")
	}
}

func TestConfigDialogueMalformedAndUnauthorizedMessagesFailClosed(t *testing.T) {
	initialize := `{"id":1,"result":{"userAgent":"x","platformFamily":"unix","platformOs":"macos","codexHome":"/synthetic"}}` + "\n"
	cases := map[string]string{
		"duplicate-key":        `{"id":1,"id":1,"result":{}}` + "\n",
		"unknown-id":           `{"id":99,"result":{}}` + "\n",
		"auth-request":         `{"id":1,"method":"account/login/start","params":{"secret":"hidden"}}` + "\n",
		"result-and-error":     `{"id":1,"result":{},"error":{"message":"synthetic-secret"}}` + "\n",
		"batch":                `[{"id":1,"result":{}}]` + "\n",
		"truncated":            `{"id":1,"result":{}}`,
		"duplicate-response":   initialize + initialize,
		"unknown-notification": `{"method":"thread/started","params":{}}` + "\n",
		"oversize":             strings.Repeat("x", (2<<20)+1) + "\n",
		"notification-flood":   strings.Repeat(`{"method":"configWarning","params":{"message":"synthetic-secret"}}`+"\n", 36),
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			i, _ := permissionFixture(t)
			var input bufferCloser
			_, err := configDialogue(&input, strings.NewReader(raw), i)
			if err == nil || strings.Contains(err.Error(), "synthetic-secret") {
				t.Fatal("invalid output accepted or leaked")
			}
			if strings.Contains(input.String(), "login") {
				t.Fatal("server action was echoed")
			}
		})
	}
}

func TestConfigInspectionStalledInputAndUncertainStopAreBounded(t *testing.T) {
	for _, uncertain := range []bool{false, true} {
		i, _ := permissionFixture(t)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		start := func(launchSpec) (process, error) {
			r, w := io.Pipe()
			return &rpcMockProcess{reader: r, writer: w, done: make(chan struct{}), uncertain: uncertain}, nil
		}
		_, _, err := inspectWithProcess(ctx, launchSpec{directory: i.Directory, inspection: &i}, start)
		cancel()
		if err == nil || errors.Is(err, errOwnership) != uncertain {
			t.Fatal("stop ownership was not preserved", err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	_, _, err := inspectWithProcess(ctx, launchSpec{}, func(launchSpec) (process, error) { calls++; return nil, nil })
	if err == nil || calls != 0 {
		t.Fatal("cancelled inspection started a child")
	}
}

type countedRPCStop struct {
	*rpcMockProcess
	stops atomic.Int32
}

func (p *countedRPCStop) Stop() error { p.stops.Add(1); return p.rpcMockProcess.Stop() }
func TestParentConfigCaptureCleansUncertainTimeoutExactlyOnce(t *testing.T) {
	i, _ := permissionFixture(t)
	r, w := io.Pipe()
	p := &countedRPCStop{rpcMockProcess: &rpcMockProcess{reader: r, writer: w, done: make(chan struct{}), uncertain: true}}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, err := inspectPermissionProcessWithStarter(ctx, launchSpec{directory: i.Directory, inspection: &i}, func(launchSpec) (process, error) { return p, nil })
	if !errors.Is(err, errOwnership) || p.stops.Load() != 1 {
		t.Fatal("timeout cleanup lost uncertainty or repeated its bounded reserve", err, p.stops.Load())
	}
}

func configCompatibilityTranscript(t *testing.T) (configInspection, string) {
	t.Helper()
	i, r := permissionFixture(t)
	initial := map[string]any{"userAgent": "synthetic", "platformFamily": "unix", "platformOs": "macos", "codexHome": "/synthetic"}
	var stream bytes.Buffer
	for n, value := range []any{initial, r, map[string]any{"requirements": nil}} {
		if err := json.NewEncoder(&stream).Encode(map[string]any{"id": n + 1, "result": value}); err != nil {
			t.Fatal(err)
		}
	}
	return i, stream.String()
}

func TestConfigPassiveNotificationsAndTypedMetadataBeforeResponseAndAfterStdinEOF(t *testing.T) {
	for _, method := range []string{"configWarning", "remoteControl/status/changed"} {
		for _, metadata := range []string{"", `,"emittedAtMs":0`, `,"emittedAtMs":9223372036854775807`, `,"emittedAtMs":-9223372036854775808`} {
			i, stream := configCompatibilityTranscript(t)
			notification := `{"method":"` + method + `","params":{"synthetic-secret":"discarded"}` + metadata + "}\n"
			var input bufferCloser
			proof, err := configDialogue(&input, strings.NewReader(notification+stream+notification), i)
			if err != nil || !validSHA256(proof.Digest) {
				t.Fatal("known passive notification rejected", err)
			}
			if strings.Contains(input.String(), "remoteControl") || strings.Contains(input.String(), "synthetic-secret") {
				t.Fatal("passive event triggered a command or escaped input")
			}
		}
	}
}

func TestConfigNotificationMetadataAndPostEOFActionsRemainStrict(t *testing.T) {
	for _, stamp := range []string{`null`, `true`, `"1"`, `1.25`, `1e3`, `9223372036854775808`, `-9223372036854775809`} {
		for _, tail := range []bool{false, true} {
			i, stream := configCompatibilityTranscript(t)
			bad := `{"method":"remoteControl/status/changed","params":{},"emittedAtMs":` + stamp + "}\n"
			if tail {
				stream += bad
			} else {
				stream = bad + stream
			}
			var input bufferCloser
			if _, err := configDialogue(&input, strings.NewReader(stream), i); err == nil {
				t.Fatal("malformed notification timestamp accepted")
			}
		}
	}
	tails := map[string]string{
		"server-request":       `{"id":99,"method":"remoteControl/status/changed","params":{}}` + "\n",
		"unknown-notification": `{"method":"remoteControl/command","params":{}}` + "\n",
		"duplicate-response":   `{"id":3,"result":{"requirements":null}}` + "\n",
		"notification-result":  `{"method":"configWarning","result":{},"params":{}}` + "\n",
		"unknown-metadata":     `{"method":"configWarning","params":{},"futureMetadata":0}` + "\n",
		"notification-flood":   strings.Repeat(`{"method":"remoteControl/status/changed","params":{},"emittedAtMs":0}`+"\n", 33),
	}
	for name, tail := range tails {
		t.Run(name, func(t *testing.T) {
			i, stream := configCompatibilityTranscript(t)
			var input bufferCloser
			if _, err := configDialogue(&input, strings.NewReader(stream+tail), i); err == nil {
				t.Fatal("post-EOF action or frame overflow accepted")
			}
		})
	}
	i, stream := configCompatibilityTranscript(t)
	stream = strings.Replace(stream, `"id":1`, `"id":1,"emittedAtMs":0`, 1)
	var input bufferCloser
	if _, err := configDialogue(&input, strings.NewReader(stream), i); err == nil {
		t.Fatal("reply metadata accepted")
	}
}

func TestConfigPassiveTailSharesOriginalTotalFrameBudget(t *testing.T) {
	for _, count := range []int{32, 33} {
		i, stream := configCompatibilityTranscript(t)
		stream += strings.Repeat(`{"method":"remoteControl/status/changed","params":{},"emittedAtMs":0}`+"\n", count)
		var input bufferCloser
		_, err := configDialogue(&input, strings.NewReader(stream), i)
		if (err == nil) != (count == 32) {
			t.Fatal("post-EOF drain changed original 35-frame limit", count, err)
		}
	}
}
