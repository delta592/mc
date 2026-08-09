// Copyright (c) 2014-2015 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

//go:build (darwin && kqueue) || (darwin && !cgo) || dragonfly || freebsd || netbsd || openbsd || solaris || illumos
// +build darwin,kqueue darwin,!cgo dragonfly freebsd netbsd openbsd solaris illumos

package fswatch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// eventTimeout bounds how long a test waits for the kernel to report a
// filesystem change. Events normally arrive in single-digit milliseconds; the
// margin is for loaded CI machines.
const eventTimeout = 5 * time.Second

// newRealWatcher returns a kqueue-backed watcher writing into the returned
// channel, closed when the test finishes.
func newRealWatcher(t *testing.T) (watcher, chan EventInfo) {
	t.Helper()
	c := make(chan EventInfo, buffer)
	w := newWatcher(c)
	if stub, ok := w.(watcherStub); ok {
		t.Skipf("kqueue watcher unavailable: %v", stub.error)
	}
	t.Cleanup(func() { _ = w.Close() })
	return w, c
}

// awaitEvent waits for an event matching want on path, ignoring unrelated
// events the filesystem may also report. It returns false on timeout.
func awaitEvent(c chan EventInfo, path string, want Event) bool {
	deadline := time.After(eventTimeout)
	for {
		select {
		case ei := <-c:
			if ei.Path() == path && ei.Event()&want != 0 {
				return true
			}
		case <-deadline:
			return false
		}
	}
}

func TestTriggerWatcherReportsCreate(t *testing.T) {
	w, c := newRealWatcher(t)
	dir := mkdirs(t)

	if err := w.Watch(dir, Create); err != nil {
		t.Fatalf("Watch() error = %v", err)
	}

	created := filepath.Join(dir, "new.txt")
	if err := os.WriteFile(created, []byte("hello"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if !awaitEvent(c, created, Create) {
		t.Fatalf("timed out waiting for Create on %q", created)
	}
}

func TestTriggerWatcherReportsRemove(t *testing.T) {
	w, c := newRealWatcher(t)
	dir := mkdirs(t)
	victim := filepath.Join(dir, "victim.txt")
	if err := os.WriteFile(victim, []byte("x"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := w.Watch(dir, Create|Remove); err != nil {
		t.Fatalf("Watch() error = %v", err)
	}
	if err := os.Remove(victim); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	if !awaitEvent(c, victim, Remove) {
		t.Fatalf("timed out waiting for Remove on %q", victim)
	}
}

func TestTriggerWatcherReportsWrite(t *testing.T) {
	w, c := newRealWatcher(t)
	dir := mkdirs(t)
	target := filepath.Join(dir, "data.txt")
	if err := os.WriteFile(target, []byte("initial"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := w.Watch(target, Write); err != nil {
		t.Fatalf("Watch() error = %v", err)
	}

	f, err := os.OpenFile(target, os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatalf("OpenFile() error = %v", err)
	}
	if _, err := f.WriteString(" more"); err != nil {
		t.Fatalf("WriteString() error = %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if !awaitEvent(c, target, Write) {
		t.Fatalf("timed out waiting for Write on %q", target)
	}
}

func TestTriggerWatcherRewatchChangesEventSet(t *testing.T) {
	w, c := newRealWatcher(t)
	dir := mkdirs(t)

	if err := w.Watch(dir, Create); err != nil {
		t.Fatalf("Watch() error = %v", err)
	}
	if err := w.Rewatch(dir, Create, Create|Remove); err != nil {
		t.Fatalf("Rewatch() error = %v", err)
	}

	victim := filepath.Join(dir, "gone.txt")
	if err := os.WriteFile(victim, []byte("x"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if !awaitEvent(c, victim, Create) {
		t.Fatalf("timed out waiting for Create on %q", victim)
	}
	if err := os.Remove(victim); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if !awaitEvent(c, victim, Remove) {
		t.Fatalf("timed out waiting for Remove after Rewatch")
	}
}

func TestTriggerWatcherUnwatchStopsEvents(t *testing.T) {
	w, c := newRealWatcher(t)
	dir := mkdirs(t)

	if err := w.Watch(dir, Create); err != nil {
		t.Fatalf("Watch() error = %v", err)
	}
	if err := w.Unwatch(dir); err != nil {
		t.Fatalf("Unwatch() error = %v", err)
	}
	// Drain anything already queued from the watch itself.
	drain(c)

	if err := os.WriteFile(filepath.Join(dir, "quiet.txt"), []byte("x"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	select {
	case ei := <-c:
		t.Fatalf("received %v on %q after Unwatch", ei.Event(), ei.Path())
	case <-time.After(300 * time.Millisecond):
	}
}

func drain(c chan EventInfo) {
	for {
		select {
		case <-c:
		default:
			return
		}
	}
}

func TestTriggerWatcherWatchMissingPath(t *testing.T) {
	w, _ := newRealWatcher(t)
	missing := filepath.Join(mkdirs(t), "missing")

	if err := w.Watch(missing, Create); err == nil {
		t.Fatal("Watch() on a missing path succeeded, want an error")
	}
	if err := w.Unwatch(missing); err == nil {
		t.Fatal("Unwatch() on a missing path succeeded, want an error")
	}
	if err := w.Rewatch(missing, Create, Remove); err == nil {
		t.Fatal("Rewatch() on a missing path succeeded, want an error")
	}
}

func TestTriggerWatcherUnwatchNotWatched(t *testing.T) {
	w, _ := newRealWatcher(t)
	dir := mkdirs(t)
	if err := w.Unwatch(dir); err == nil {
		t.Fatal("Unwatch() on an unwatched path succeeded, want an error")
	}
}

// TestTriggerWatcherDoubleWatchIsIdempotent checks that re-watching a path is
// accepted rather than rejected, and that events still flow afterwards.
func TestTriggerWatcherDoubleWatchIsIdempotent(t *testing.T) {
	w, c := newRealWatcher(t)
	dir := mkdirs(t)

	if err := w.Watch(dir, Create); err != nil {
		t.Fatalf("Watch() error = %v", err)
	}
	if err := w.Watch(dir, Create); err != nil {
		t.Fatalf("second Watch() error = %v, want it to be accepted", err)
	}

	created := filepath.Join(dir, "still-works.txt")
	if err := os.WriteFile(created, []byte("x"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if !awaitEvent(c, created, Create) {
		t.Fatalf("timed out waiting for Create on %q after a repeated Watch", created)
	}
}

func TestTriggerWatcherWatchesSubdirectoryContents(t *testing.T) {
	w, c := newRealWatcher(t)
	dir := mkdirs(t, "sub")
	sub := filepath.Join(dir, "sub")

	if err := w.Watch(sub, Create); err != nil {
		t.Fatalf("Watch() error = %v", err)
	}
	created := filepath.Join(sub, "inner.txt")
	if err := os.WriteFile(created, []byte("x"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if !awaitEvent(c, created, Create) {
		t.Fatalf("timed out waiting for Create on %q", created)
	}
}

func TestTriggerWatcherCloseIsIdempotentlySafe(t *testing.T) {
	c := make(chan EventInfo, buffer)
	w := newWatcher(c)
	if stub, ok := w.(watcherStub); ok {
		t.Skipf("kqueue watcher unavailable: %v", stub.error)
	}
	dir := mkdirs(t)
	if err := w.Watch(dir, Create); err != nil {
		t.Fatalf("Watch() error = %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

// TestEndToEndWatchViaPublicAPI drives the package-level Watch/Stop entry
// points, which run against the process-wide default tree.
func TestEndToEndWatchViaPublicAPI(t *testing.T) {
	dir := mkdirs(t)
	c := make(chan EventInfo, buffer)

	if err := Watch(dir, c, Create); err != nil {
		t.Fatalf("Watch() error = %v", err)
	}
	defer Stop(c)

	created := filepath.Join(dir, "public.txt")
	if err := os.WriteFile(created, []byte("x"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if !awaitEvent(c, created, Create) {
		t.Fatalf("timed out waiting for Create on %q", created)
	}
}

// TestEndToEndRecursiveWatchViaPublicAPI checks the recursive "..." form finds
// events in nested directories.
func TestEndToEndRecursiveWatchViaPublicAPI(t *testing.T) {
	dir := mkdirs(t, filepath.Join("a", "b"))
	nested := filepath.Join(dir, "a", "b")
	c := make(chan EventInfo, buffer)

	if err := Watch(dir+"...", c, Create); err != nil {
		t.Fatalf("Watch() error = %v", err)
	}
	defer Stop(c)

	created := filepath.Join(nested, "deep.txt")
	if err := os.WriteFile(created, []byte("x"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if !awaitEvent(c, created, Create) {
		t.Fatalf("timed out waiting for Create on %q", created)
	}
}

// TestEndToEndStopEndsDelivery checks Stop detaches the channel from the
// default tree.
func TestEndToEndStopEndsDelivery(t *testing.T) {
	dir := mkdirs(t)
	c := make(chan EventInfo, buffer)

	if err := Watch(dir, c, Create); err != nil {
		t.Fatalf("Watch() error = %v", err)
	}
	Stop(c)
	drain(c)

	if err := os.WriteFile(filepath.Join(dir, "after.txt"), []byte("x"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	select {
	case ei := <-c:
		t.Fatalf("received %v on %q after Stop", ei.Event(), ei.Path())
	case <-time.After(300 * time.Millisecond):
	}
}

// TestTriggerEventInfo covers the kqueue/FEN event type's EventInfo
// implementation, including its fmt.Stringer.
func TestTriggerEventInfo(t *testing.T) {
	e := &event{p: "/tmp/x", e: Create, d: true}
	if e.Path() != "/tmp/x" {
		t.Fatalf("Path() = %q, want /tmp/x", e.Path())
	}
	if e.Event() != Create {
		t.Fatalf("Event() = %v, want Create", e.Event())
	}
	if e.Sys() != nil {
		t.Fatalf("Sys() = %v, want nil", e.Sys())
	}
	isdir, err := e.isDir()
	if err != nil || !isdir {
		t.Fatalf("isDir() = %v, %v, want true, nil", isdir, err)
	}
	if got := e.String(); !strings.Contains(got, "/tmp/x") {
		t.Fatalf("String() = %q, want it to mention the path", got)
	}
}
