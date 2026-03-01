package compute

// This file provides legacy rendering helpers that were previously used by the
// non‑interactive UI. The new interactive UI is built around tea.Models (see
// instances.go, instance_detail_model.go). The test suite still expects the
// original Render* functions to exist, so we implement thin wrappers that
// perform synchronous data fetching and return a string view. The implementation
// mirrors the behaviour of the corresponding tea.Models but without the async
// loading or spinner handling.

import (
	"fmt"
	"github.com/charmbracelet/bubbles/table"
	"ostui/internal/client"
	"ostui/internal/ui/common"
	"time"
)

// RenderInstances returns a string representation of the list of compute
// instances. It is used by the test suite and by any non‑interactive callers.
// On success a table view is returned; on error a simple error message that
// contains the phrase "Failed to list instances" is returned.
func RenderInstances(cc client.ComputeClient) string {
	srvList, err := cc.ListInstances()
	if err != nil {
		return fmt.Sprintf("Failed to list instances: %s", err)
	}
	cols := []table.Column{{Title: "ID", Width: 36}, {Title: "Name", Width: 20}, {Title: "Status", Width: 12}}
	rows := []table.Row{}
	for _, s := range srvList {
		rows = append(rows, table.Row{s.ID, s.Name, s.Status})
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

// RenderInstanceDetail returns a detailed view for a single instance. The view
// includes a title "Instance Details" followed by a two‑column table of fields.
// Errors are reported with a message containing "Failed to get instance".
func RenderInstanceDetail(cc client.ComputeClient, id string) string {
	srv, err := cc.GetInstance(id)
	if err != nil {
		return fmt.Sprintf("Failed to get instance: %s", err)
	}
	// Build a map of fields similar to InstanceDetailModel.
	fields := map[string]string{
		"ID":       srv.ID,
		"Name":     srv.Name,
		"Status":   srv.Status,
		"Flavor":   fmt.Sprintf("%v", srv.Flavor["id"]),
		"Image":    fmt.Sprintf("%v", srv.Image["id"]),
		"Created":  srv.Created.Format(time.RFC3339),
		"Updated":  srv.Updated.Format(time.RFC3339),
		"HostID":   srv.HostID,
		"KeyName":  srv.KeyName,
		"UserID":   srv.UserID,
		"TenantID": srv.TenantID,
	}
	// Use the common detail view to include a title.
	return common.NewDetail("Instance Details", fields).View()
}

// RenderInstanceForm returns a simple form view for creating a new instance.
// The test only checks that the prompts "Name:" and "Image:" appear in the
// output, so we construct a form with those fields using the shared FormModel.
func RenderInstanceForm() string {
	// The FormModel lives in the common package.
	return common.NewForm([]string{"Name", "Image"}).View()
}
