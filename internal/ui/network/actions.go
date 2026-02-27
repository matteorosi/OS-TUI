package network

import (
	"fmt"
	"github.com/charmbracelet/bubbles/table"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/floatingips"
	"ostui/internal/client"
	"ostui/internal/ui/common"
)

// AllocateFloatingIP allocates a new floating IP on the given network ID.
// Returns a string view indicating success or error.
func AllocateFloatingIP(nc client.NetworkClient, networkID string) string {
	opts := floatingips.CreateOpts{FloatingNetworkID: networkID}
	fip, err := nc.AllocateFloatingIP(opts)
	if err != nil {
		cols := []table.Column{{Title: "Error", Width: 80}}
		rows := []table.Row{{"Failed to allocate floating IP: " + err.Error()}}
		return common.NewTable(cols, rows).View()
	}
	// Show allocated floating IP details.
	cols := []table.Column{{Title: "ID", Width: 36}, {Title: "FloatingNetworkID", Width: 36}, {Title: "FixedIP", Width: 15}, {Title: "PortID", Width: 36}, {Title: "Status", Width: 12}}
	rows := []table.Row{{fip.ID, fip.FloatingNetworkID, fip.FixedIP, fip.PortID, fip.Status}}
	return common.NewTable(cols, rows).View()
}

// ReleaseFloatingIP releases (deletes) a floating IP by its ID.
// Returns a string view indicating success or error.
func ReleaseFloatingIP(nc client.NetworkClient, fipID string) string {
	err := nc.ReleaseFloatingIP(fipID)
	if err != nil {
		cols := []table.Column{{Title: "Error", Width: 80}}
		rows := []table.Row{{"Failed to release floating IP: " + err.Error()}}
		return common.NewTable(cols, rows).View()
	}
	return fmt.Sprintf("Floating IP %s released successfully.", fipID)
}

// AssociateFloatingIP associates a floating IP with a port.
// Returns a string view of the updated floating IP or error.
func AssociateFloatingIP(nc client.NetworkClient, fipID, portID string) string {
	fip, err := nc.AssociateFloatingIP(fipID, portID)
	if err != nil {
		cols := []table.Column{{Title: "Error", Width: 80}}
		rows := []table.Row{{"Failed to associate floating IP: " + err.Error()}}
		return common.NewTable(cols, rows).View()
	}
	cols := []table.Column{{Title: "ID", Width: 36}, {Title: "FloatingNetworkID", Width: 36}, {Title: "FixedIP", Width: 15}, {Title: "PortID", Width: 36}, {Title: "Status", Width: 12}}
	rows := []table.Row{{fip.ID, fip.FloatingNetworkID, fip.FixedIP, fip.PortID, fip.Status}}
	return common.NewTable(cols, rows).View()
}

// DisassociateFloatingIP removes any port association from a floating IP.
// Returns a string view of the updated floating IP or error.
func DisassociateFloatingIP(nc client.NetworkClient, fipID string) string {
	fip, err := nc.DisassociateFloatingIP(fipID)
	if err != nil {
		cols := []table.Column{{Title: "Error", Width: 80}}
		rows := []table.Row{{"Failed to disassociate floating IP: " + err.Error()}}
		return common.NewTable(cols, rows).View()
	}
	cols := []table.Column{{Title: "ID", Width: 36}, {Title: "FloatingNetworkID", Width: 36}, {Title: "FixedIP", Width: 15}, {Title: "PortID", Width: 36}, {Title: "Status", Width: 12}}
	rows := []table.Row{{fip.ID, fip.FloatingNetworkID, fip.FixedIP, fip.PortID, fip.Status}}
	return common.NewTable(cols, rows).View()
}
