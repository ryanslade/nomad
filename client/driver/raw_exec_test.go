package driver

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/hashicorp/nomad/client/config"
	"github.com/hashicorp/nomad/client/driver/env"
	"github.com/hashicorp/nomad/helper/testtask"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/hashicorp/nomad/testutil"
)

func TestRawExecDriver_Fingerprint(t *testing.T) {
	t.Parallel()
	driverCtx, _ := testDriverContexts(&structs.Task{Name: "foo"})
	d := NewRawExecDriver(driverCtx)
	node := &structs.Node{
		Attributes: make(map[string]string),
	}

	// Disable raw exec.
	cfg := &config.Config{Options: map[string]string{rawExecConfigOption: "false"}}

	apply, err := d.Fingerprint(cfg, node)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if apply {
		t.Fatalf("should not apply")
	}
	if node.Attributes["driver.raw_exec"] != "" {
		t.Fatalf("driver incorrectly enabled")
	}

	// Enable raw exec.
	cfg.Options[rawExecConfigOption] = "true"
	apply, err = d.Fingerprint(cfg, node)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !apply {
		t.Fatalf("should apply")
	}
	if node.Attributes["driver.raw_exec"] != "1" {
		t.Fatalf("driver not enabled")
	}
}

func TestRawExecDriver_StartOpen_Wait(t *testing.T) {
	t.Parallel()
	task := &structs.Task{
		Name: "sleep",
		Config: map[string]interface{}{
			"command": testtask.Path(),
			"args":    []string{"sleep", "1s"},
		},
		Resources: basicResources,
	}
	testtask.SetTaskEnv(task)
	driverCtx, execCtx := testDriverContexts(task)
	defer execCtx.AllocDir.Destroy()
	d := NewRawExecDriver(driverCtx)

	handle, err := d.Start(execCtx, task)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if handle == nil {
		t.Fatalf("missing handle")
	}

	// Attempt to open
	handle2, err := d.Open(execCtx, handle.ID())
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if handle2 == nil {
		t.Fatalf("missing handle")
	}

	// Task should terminate quickly
	select {
	case <-handle2.WaitCh():
	case <-time.After(time.Duration(testutil.TestMultiplier()*5) * time.Second):
		t.Fatalf("timeout")
	}
}

func TestRawExecDriver_Start_Artifact_basic(t *testing.T) {
	t.Parallel()
	path := testtask.Path()
	ts := httptest.NewServer(http.FileServer(http.Dir(filepath.Dir(path))))
	defer ts.Close()

	file := filepath.Base(path)
	task := &structs.Task{
		Name: "sleep",
		Config: map[string]interface{}{
			"artifact_source": fmt.Sprintf("%s/%s", ts.URL, file),
			"command":         filepath.Join("$NOMAD_TASK_DIR", file),
			"args":            []string{"sleep", "1s"},
		},
		Resources: basicResources,
	}
	testtask.SetTaskEnv(task)

	driverCtx, execCtx := testDriverContexts(task)
	defer execCtx.AllocDir.Destroy()
	d := NewRawExecDriver(driverCtx)

	handle, err := d.Start(execCtx, task)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if handle == nil {
		t.Fatalf("missing handle")
	}

	// Attempt to open
	handle2, err := d.Open(execCtx, handle.ID())
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if handle2 == nil {
		t.Fatalf("missing handle")
	}

	// Task should terminate quickly
	select {
	case <-handle2.WaitCh():
	case <-time.After(time.Duration(testutil.TestMultiplier()*5) * time.Second):
		t.Fatalf("timeout")
	}
}

func TestRawExecDriver_Start_Artifact_expanded(t *testing.T) {
	t.Parallel()
	path := testtask.Path()
	ts := httptest.NewServer(http.FileServer(http.Dir(filepath.Dir(path))))
	defer ts.Close()

	file := filepath.Base(path)
	task := &structs.Task{
		Name: "sleep",
		Config: map[string]interface{}{
			"artifact_source": fmt.Sprintf("%s/%s", ts.URL, file),
			"command":         filepath.Join("$NOMAD_TASK_DIR", file),
			"args":            []string{"sleep", "1s"},
		},
		Resources: basicResources,
	}
	testtask.SetTaskEnv(task)

	driverCtx, execCtx := testDriverContexts(task)
	defer execCtx.AllocDir.Destroy()
	d := NewRawExecDriver(driverCtx)

	handle, err := d.Start(execCtx, task)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if handle == nil {
		t.Fatalf("missing handle")
	}

	// Attempt to open
	handle2, err := d.Open(execCtx, handle.ID())
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if handle2 == nil {
		t.Fatalf("missing handle")
	}

	// Task should terminate quickly
	select {
	case <-handle2.WaitCh():
	case <-time.After(time.Duration(testutil.TestMultiplier()*5) * time.Second):
		t.Fatalf("timeout")
	}
}

func TestRawExecDriver_Start_Wait(t *testing.T) {
	t.Parallel()
	task := &structs.Task{
		Name: "sleep",
		Config: map[string]interface{}{
			"command": testtask.Path(),
			"args":    []string{"sleep", "1s"},
		},
		Resources: basicResources,
	}
	testtask.SetTaskEnv(task)
	driverCtx, execCtx := testDriverContexts(task)
	defer execCtx.AllocDir.Destroy()
	d := NewRawExecDriver(driverCtx)

	handle, err := d.Start(execCtx, task)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if handle == nil {
		t.Fatalf("missing handle")
	}

	// Update should be a no-op
	err = handle.Update(task)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Task should terminate quickly
	select {
	case res := <-handle.WaitCh():
		if !res.Successful() {
			t.Fatalf("err: %v", res)
		}
	case <-time.After(time.Duration(testutil.TestMultiplier()*5) * time.Second):
		t.Fatalf("timeout")
	}
}

func TestRawExecDriver_Start_Wait_AllocDir(t *testing.T) {
	t.Parallel()
	exp := []byte{'w', 'i', 'n'}
	file := "output.txt"
	outPath := fmt.Sprintf(`$%s/%s`, env.AllocDir, file)
	task := &structs.Task{
		Name: "sleep",
		Config: map[string]interface{}{
			"command": testtask.Path(),
			"args": []string{
				"sleep", "1s",
				"write", string(exp), outPath,
			},
		},
		Resources: basicResources,
	}
	testtask.SetTaskEnv(task)

	driverCtx, execCtx := testDriverContexts(task)
	defer execCtx.AllocDir.Destroy()
	d := NewRawExecDriver(driverCtx)

	handle, err := d.Start(execCtx, task)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if handle == nil {
		t.Fatalf("missing handle")
	}

	// Task should terminate quickly
	select {
	case res := <-handle.WaitCh():
		if !res.Successful() {
			t.Fatalf("err: %v", res)
		}
	case <-time.After(time.Duration(testutil.TestMultiplier()*5) * time.Second):
		t.Fatalf("timeout")
	}

	// Check that data was written to the shared alloc directory.
	outputFile := filepath.Join(execCtx.AllocDir.SharedDir, file)
	act, err := ioutil.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Couldn't read expected output: %v", err)
	}

	if !reflect.DeepEqual(act, exp) {
		t.Fatalf("Command outputted %v; want %v", act, exp)
	}
}

func TestRawExecDriver_Start_Kill_Wait(t *testing.T) {
	t.Parallel()
	task := &structs.Task{
		Name: "sleep",
		Config: map[string]interface{}{
			"command": testtask.Path(),
			"args":    []string{"sleep", "15s"},
		},
		Resources: basicResources,
	}
	testtask.SetTaskEnv(task)

	driverCtx, execCtx := testDriverContexts(task)
	defer execCtx.AllocDir.Destroy()
	d := NewRawExecDriver(driverCtx)

	handle, err := d.Start(execCtx, task)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if handle == nil {
		t.Fatalf("missing handle")
	}

	go func() {
		time.Sleep(1 * time.Second)
		err := handle.Kill()

		// Can't rely on the ordering between wait and kill on travis...
		if !testutil.IsTravis() && err != nil {
			t.Fatalf("err: %v", err)
		}
	}()

	// Task should terminate quickly
	select {
	case res := <-handle.WaitCh():
		if res.Successful() {
			t.Fatal("should err")
		}
	case <-time.After(time.Duration(testutil.TestMultiplier()*5) * time.Second):
		t.Fatalf("timeout")
	}
}
