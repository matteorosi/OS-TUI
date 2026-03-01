package storage

import (
	"fmt"
	"github.com/charmbracelet/bubbles/table"
	"ostui/internal/client"
	"ostui/internal/ui/common"
)

// RenderVolumes returns a string representation of the list of storage volumes.
func RenderVolumes(sc client.StorageClient) string {
	volList, err := sc.ListVolumes()
	if err != nil {
		return fmt.Sprintf("Failed to list volumes: %s", err)
	}
	cols := []table.Column{{Title: "ID", Width: 36}, {Title: "Name", Width: 20}, {Title: "Size", Width: 8}, {Title: "Status", Width: 12}}
	rows := []table.Row{}
	for _, v := range volList {
		rows = append(rows, table.Row{v.ID, v.Name, fmt.Sprintf("%d", v.Size), v.Status})
	}
	t := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(20),
	)
	t.SetStyles(table.DefaultStyles())
	return t.View()
}

// RenderVolumeDetail returns a detailed view for a specific volume.
func RenderVolumeDetail(sc client.StorageClient, volumeID string) string {
	vol, err := sc.GetVolume(volumeID)
	if err != nil {
		return fmt.Sprintf("Failed to get volume: %s", err)
	}
	fields := map[string]string{
		"ID":          vol.ID,
		"Name":        vol.Name,
		"Size":        fmt.Sprintf("%d", vol.Size),
		"Status":      vol.Status,
		"Description": vol.Description,
	}
	return common.NewDetail("Volume Details", fields).View()
}

// RenderSnapshots returns a string representation of the list of snapshots.
func RenderSnapshots(sc client.StorageClient) string {
	snapList, err := sc.ListSnapshots()
	if err != nil {
		return fmt.Sprintf("Failed to list snapshots: %s", err)
	}
	cols := []table.Column{{Title: "ID", Width: 36}, {Title: "Name", Width: 20}, {Title: "VolumeID", Width: 36}, {Title: "Size", Width: 8}, {Title: "Status", Width: 12}, {Title: "Created", Width: 20}}
	rows := []table.Row{}
	for _, snap := range snapList {
		rows = append(rows, table.Row{snap.ID, snap.Name, snap.VolumeID, fmt.Sprintf("%d", snap.Size), snap.Status, snap.CreatedAt.Format("2006-01-02 15:04:05")})
	}
	t := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(20),
	)
	t.SetStyles(table.DefaultStyles())
	return t.View()
}

// RenderBuckets returns a string representation of the list of object storage buckets.
func RenderBuckets(osc client.ObjectStorageClient) string {
	bucketList, err := osc.ListBuckets()
	if err != nil {
		return fmt.Sprintf("Failed to list buckets: %s", err)
	}
	cols := []table.Column{{Title: "Name", Width: 20}, {Title: "Count", Width: 8}, {Title: "Bytes", Width: 12}}
	rows := []table.Row{}
	for _, b := range bucketList {
		rows = append(rows, table.Row{b.Name, fmt.Sprintf("%d", b.Count), fmt.Sprintf("%d", b.Bytes)})
	}
	t := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(20),
	)
	t.SetStyles(table.DefaultStyles())
	return t.View()
}
