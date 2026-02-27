package storage

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/snapshots"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/openstack/objectstorage/v1/containers"
)

type mockStorageClient struct {
	volumes []volumes.Volume
	volErr  error

	volume volumes.Volume
	getErr error

	deleteErr error

	snapshots []snapshots.Snapshot
	snapErr   error

	createdSnapshot snapshots.Snapshot
	createSnapErr   error
}

func (m *mockStorageClient) ListVolumes() ([]volumes.Volume, error) {
	return m.volumes, m.volErr
}
func (m *mockStorageClient) GetVolume(id string) (volumes.Volume, error) {
	return m.volume, m.getErr
}
func (m *mockStorageClient) DeleteVolume(id string) error {
	return m.deleteErr
}
func (m *mockStorageClient) ListSnapshots() ([]snapshots.Snapshot, error) {
	return m.snapshots, m.snapErr
}
func (m *mockStorageClient) CreateSnapshot(opts snapshots.CreateOptsBuilder) (snapshots.Snapshot, error) {
	return m.createdSnapshot, m.createSnapErr
}

type mockObjectStorageClient struct {
	buckets   []containers.Container
	bucketErr error
}

func (m *mockObjectStorageClient) ListBuckets() ([]containers.Container, error) {
	return m.buckets, m.bucketErr
}

func TestRenderVolumesSuccess(t *testing.T) {
	mock := &mockStorageClient{volumes: []volumes.Volume{{ID: "vol-1", Name: "vol1", Size: 10, Status: "available"}}}
	out := RenderVolumes(mock)
	if !strings.Contains(out, "vol1") {
		t.Fatalf("expected volume name in output, got %s", out)
	}
}

func TestRenderVolumesError(t *testing.T) {
	mock := &mockStorageClient{volErr: errors.New("list error")}
	out := RenderVolumes(mock)
	if !strings.Contains(out, "Failed to list volumes") {
		t.Fatalf("expected error message, got %s", out)
	}
}

func TestRenderVolumeDetailSuccess(t *testing.T) {
	mock := &mockStorageClient{volume: volumes.Volume{ID: "vol-1", Name: "vol1", Size: 10, Status: "available", CreatedAt: time.Now(), UpdatedAt: time.Now(), Description: "test volume", VolumeType: "ssd", Bootable: "true"}}
	out := RenderVolumeDetail(mock, "vol-1")
	if !strings.Contains(out, "Volume Details") {
		t.Fatalf("expected detail title, got %s", out)
	}
	if !strings.Contains(out, "vol1") {
		t.Fatalf("expected volume name, got %s", out)
	}
}

func TestRenderVolumeDetailError(t *testing.T) {
	mock := &mockStorageClient{getErr: errors.New("get error")}
	out := RenderVolumeDetail(mock, "vol-1")
	if !strings.Contains(out, "Failed to get volume") {
		t.Fatalf("expected error message, got %s", out)
	}
}

func TestRenderSnapshotsSuccess(t *testing.T) {
	mock := &mockStorageClient{snapshots: []snapshots.Snapshot{{ID: "snap-1", Name: "snap1", VolumeID: "vol-1", Size: 10, Status: "available", CreatedAt: time.Now()}}}
	out := RenderSnapshots(mock)
	if !strings.Contains(out, "snap1") {
		t.Fatalf("expected snapshot name in output, got %s", out)
	}
}

func TestRenderSnapshotsError(t *testing.T) {
	mock := &mockStorageClient{snapErr: errors.New("list error")}
	out := RenderSnapshots(mock)
	if !strings.Contains(out, "Failed to list snapshots") {
		t.Fatalf("expected error message, got %s", out)
	}
}

func TestRenderBucketsSuccess(t *testing.T) {
	mock := &mockObjectStorageClient{buckets: []containers.Container{{Name: "bucket1", Count: 5, Bytes: 1024}}}
	out := RenderBuckets(mock)
	if !strings.Contains(out, "bucket1") {
		t.Fatalf("expected bucket name in output, got %s", out)
	}
}

func TestRenderBucketsError(t *testing.T) {
	mock := &mockObjectStorageClient{bucketErr: errors.New("list error")}
	out := RenderBuckets(mock)
	if !strings.Contains(out, "Failed to list buckets") {
		t.Fatalf("expected error message, got %s", out)
	}
}

func TestDeleteVolumeSuccess(t *testing.T) {
	mock := &mockStorageClient{deleteErr: nil}
	out := DeleteVolume(mock, "vol-1")
	if !strings.Contains(out, "deleted successfully") {
		t.Fatalf("expected success message, got %s", out)
	}
}

func TestDeleteVolumeError(t *testing.T) {
	mock := &mockStorageClient{deleteErr: errors.New("delete error")}
	out := DeleteVolume(mock, "vol-1")
	if !strings.Contains(out, "Failed to delete volume") {
		t.Fatalf("expected error message, got %s", out)
	}
}

func TestCreateSnapshotSuccess(t *testing.T) {
	mock := &mockStorageClient{createdSnapshot: snapshots.Snapshot{ID: "snap-1", Name: "snap1", VolumeID: "vol-1", Status: "available", CreatedAt: time.Now()}}
	out := CreateSnapshot(mock, "vol-1", "snap1")
	if !strings.Contains(out, "snap1") {
		t.Fatalf("expected snapshot name in output, got %s", out)
	}
}

func TestCreateSnapshotError(t *testing.T) {
	mock := &mockStorageClient{createSnapErr: errors.New("create error")}
	out := CreateSnapshot(mock, "vol-1", "snap1")
	if !strings.Contains(out, "Failed to create snapshot") {
		t.Fatalf("expected error message, got %s", out)
	}
}
