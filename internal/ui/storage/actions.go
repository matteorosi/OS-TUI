package storage

import (
	"fmt"
	"github.com/charmbracelet/bubbles/table"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/snapshots"
	"ostui/internal/client"
	"ostui/internal/ui/common"
	"ostui/internal/ui/uiconst"
)

// DeleteVolume deletes a volume by its ID using the provided StorageClient.
// Returns a string view indicating success or an error table.
func DeleteVolume(sc client.StorageClient, volumeID string) string {
	err := sc.DeleteVolume(volumeID)
	if err != nil {
		cols := []table.Column{{Title: "Error", Width: uiconst.ColWidthError}}
		rows := []table.Row{{"Failed to delete volume: " + err.Error()}}
		return common.NewTable(cols, rows).View()
	}
	return fmt.Sprintf("Volume %s deleted successfully.", volumeID)
}

// CreateSnapshot creates a snapshot for the given volume ID with the provided name.
// Returns a string view of the created snapshot or an error table.
func CreateSnapshot(sc client.StorageClient, volumeID, name string) string {
	opts := snapshots.CreateOpts{VolumeID: volumeID, Name: name}
	snap, err := sc.CreateSnapshot(opts)
	if err != nil {
		cols := []table.Column{{Title: "Error", Width: uiconst.ColWidthError}}
		rows := []table.Row{{"Failed to create snapshot: " + err.Error()}}
		return common.NewTable(cols, rows).View()
	}
	// Show snapshot details in a table.
	cols := []table.Column{{Title: "ID", Width: uiconst.ColWidthUUID}, {Title: "Name", Width: uiconst.ColWidthName}, {Title: "VolumeID", Width: uiconst.ColWidthUUID}, {Title: "Status", Width: uiconst.ColWidthStatus}, {Title: "Created", Width: uiconst.ColWidthField}}
	rows := []table.Row{{snap.ID, snap.Name, snap.VolumeID, snap.Status, snap.CreatedAt.Format("2006-01-02 15:04:05")}}
	return common.NewTable(cols, rows).View()
}
