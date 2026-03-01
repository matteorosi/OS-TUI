package network

import (
	"fmt"
	"github.com/charmbracelet/bubbles/table"
	"ostui/internal/client"
	"ostui/internal/ui/common"
)

// RenderNetworks returns a string representation of the list of networks.
func RenderNetworks(nc client.NetworkClient) string {
	netList, err := nc.ListNetworks()
	if err != nil {
		return fmt.Sprintf("Failed to list networks: %s", err)
	}
	cols := []table.Column{{Title: "ID", Width: 36}, {Title: "Name", Width: 20}, {Title: "Status", Width: 12}}
	rows := []table.Row{}
	for _, n := range netList {
		rows = append(rows, table.Row{n.ID, n.Name, n.Status})
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

// RenderSubnets returns a string representation of the list of subnets.
func RenderSubnets(nc client.NetworkClient) string {
	subList, err := nc.ListSubnets()
	if err != nil {
		return fmt.Sprintf("Failed to list subnets: %s", err)
	}
	cols := []table.Column{{Title: "ID", Width: 36}, {Title: "Name", Width: 20}, {Title: "CIDR", Width: 20}, {Title: "IPVer", Width: 6}}
	rows := []table.Row{}
	for _, s := range subList {
		rows = append(rows, table.Row{s.ID, s.Name, s.CIDR, fmt.Sprintf("%d", s.IPVersion)})
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

// RenderFloatingIPs returns a string representation of the list of floating IPs.
func RenderFloatingIPs(nc client.NetworkClient) string {
	fipList, err := nc.ListFloatingIPs()
	if err != nil {
		return fmt.Sprintf("Failed to list floating IPs: %s", err)
	}
	cols := []table.Column{{Title: "ID", Width: 36}, {Title: "FloatingNetworkID", Width: 36}, {Title: "FixedIP", Width: 15}, {Title: "PortID", Width: 36}, {Title: "Status", Width: 12}}
	rows := []table.Row{}
	for _, f := range fipList {
		rows = append(rows, table.Row{f.ID, f.FloatingNetworkID, f.FixedIP, f.PortID, f.Status})
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

// RenderSecurityGroups returns a string representation of the list of security groups.
func RenderSecurityGroups(nc client.NetworkClient) string {
	sgList, err := nc.ListSecurityGroups()
	if err != nil {
		return fmt.Sprintf("Failed to list security groups: %s", err)
	}
	cols := []table.Column{{Title: "ID", Width: 36}, {Title: "Name", Width: 20}, {Title: "Description", Width: 30}, {Title: "Stateful", Width: 8}}
	rows := []table.Row{}
	for _, sg := range sgList {
		rows = append(rows, table.Row{sg.ID, sg.Name, sg.Description, fmt.Sprintf("%v", sg.Stateful)})
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

// RenderSecurityGroupDetail returns a detailed view for a specific security group.
func RenderSecurityGroupDetail(nc client.NetworkClient, sgID string) string {
	sgList, err := nc.ListSecurityGroups()
	if err != nil {
		return fmt.Sprintf("Failed to list security groups: %s", err)
	}
	var sg *struct {
		ID          string
		Name        string
		Description string
		Stateful    bool
	}
	for _, g := range sgList {
		if g.ID == sgID {
			sg = &struct {
				ID          string
				Name        string
				Description string
				Stateful    bool
			}{ID: g.ID, Name: g.Name, Description: g.Description, Stateful: g.Stateful}
			break
		}
	}
	if sg == nil {
		return "Security group not found"
	}
	fields := map[string]string{
		"ID":          sg.ID,
		"Name":        sg.Name,
		"Description": sg.Description,
		"Stateful":    fmt.Sprintf("%v", sg.Stateful),
	}
	return common.NewDetail("Security Group Details", fields).View()
}

// RenderAllocateFloatingIPForm returns a simple form for allocating a floating IP.
func RenderAllocateFloatingIPForm() string {
	// The form only requires the FloatingNetworkID field for the test.
	return common.NewForm([]string{"FloatingNetworkID"}).View()
}
