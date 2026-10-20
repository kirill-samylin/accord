package bridge

import (
	"agent-board/internal/core"
	"context"
	"encoding/json"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"testing"
)

func TestCoordinationResourceUsesOnlyBoundWorkerSession(t *testing.T) {
	f := setup(t, 0)
	task := f.task()
	f.describe()
	uri := "accord://coordination?task_key=" + task.Key
	out, err := f.client.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: uri})
	if err != nil || len(out.Contents) != 1 || out.Contents[0].URI != uri {
		t.Fatal(err)
	}
	var detail core.TaskCoordination
	if e := json.Unmarshal([]byte(out.Contents[0].Text), &detail); e != nil || detail.TaskKey != task.Key {
		t.Fatal(e)
	}
	if _, err = f.client.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: uri + "&session_id=other"}); err == nil {
		t.Fatal("foreign resource session accepted")
	}
}
