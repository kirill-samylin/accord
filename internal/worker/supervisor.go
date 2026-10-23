package worker

import (
	"encoding/json"
	"errors"
	"io"
)

type supervisorRequest struct {
	Executable string   `json:"executable"`
	Directory  string   `json:"directory"`
	Args       []string `json:"args"`
	Env        []string `json:"env"`
	Prompt     string   `json:"prompt"`
	Nonce      string   `json:"nonce"`
}

type supervisorProof struct {
	Type       string `json:"type"`
	Nonce      string `json:"nonce"`
	Proven     bool   `json:"proven"`
	Success    bool   `json:"success"`
	Forced     bool   `json:"forced"`
	Diagnostic string `json:"local_diagnostic,omitempty"`
	ExitCode   *int   `json:"exit_code,omitempty"`
}

// Supervise lives in a separate OS process. Its stdin is a private parent
// lifeline, never the provider's stdin: EOF also occurs after SIGKILL/crash.
func Supervise(input io.Reader, output io.Writer) error {
	decoder := json.NewDecoder(io.LimitReader(input, 256<<10))
	decoder.DisallowUnknownFields()
	var request supervisorRequest
	if err := decoder.Decode(&request); err != nil || !identifier.MatchString(request.Nonce) {
		return errors.New("invalid local supervisor handshake")
	}
	child, err := startProcess(launchSpec{executable: request.Executable, directory: request.Directory, args: request.Args, env: request.Env, prompt: request.Prompt})
	proof := supervisorProof{Type: "accord.supervisor.exit", Nonce: request.Nonce, Proven: true}
	if err != nil {
		return json.NewEncoder(output).Encode(proof)
	}
	go func() {
		// Include bytes buffered by Decode so the private framing cannot lose
		// EOF. Providers get a different pipe; no native descendant owns this FD.
		_, _ = io.Copy(io.Discard, io.MultiReader(decoder.Buffered(), input))
		_ = child.Stop()
	}()
	_, copyError := io.Copy(output, child.Output())
	child.Output().Close()
	if copyError != nil {
		_ = child.Stop()
	}
	exitError := child.Wait()
	proof.Proven = !errors.Is(exitError, errOwnership)
	proof.Diagnostic = ownershipDiagnostic(exitError)
	proof.Success = exitError == nil && copyError == nil
	proof.Forced = child.Forced()
	if code, ok := processExitCode(child); ok {
		proof.ExitCode = &code
	}
	// Keep the final proof on its own JSONL boundary even if the native
	// provider ended an incomplete line. No native output can guess Nonce.
	if _, err = io.WriteString(output, "\n"); err != nil {
		return err
	}
	return json.NewEncoder(output).Encode(proof)
}
