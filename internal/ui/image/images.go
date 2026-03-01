package image

import (
	"context"
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"ostui/internal/client"
	"ostui/internal/ui/uiconst"
	"strings"
)

// ImagesModel implements a subview for listing OpenStack images.
type ImagesModel struct {
	table      table.Model
	loading    bool
	err        error
	spinner    spinner.Model
	client     client.ImageClient
	allRows    []table.Row
	filterMode bool
	filter     textinput.Model
	// Dynamic sizing
	width  int
	height int
}

// NewImagesModel creates a new ImagesModel with the given image client.
func NewImagesModel(ic client.ImageClient) ImagesModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	ti := textinput.New()
	ti.Placeholder = "filter..."
	// Initialize with reasonable defaults.
	return ImagesModel{client: ic, loading: true, spinner: s, filter: ti, width: 120, height: 30}
}

type imagesDataLoadedMsg struct {
	tbl  table.Model
	rows []table.Row
	err  error
}

// Init starts async loading of images.
func (m ImagesModel) Init() tea.Cmd {
	return func() tea.Msg {
		imgList, err := m.client.ListImages(context.Background())
		if err != nil {
			return imagesDataLoadedMsg{err: err}
		}
		cols := []table.Column{{Title: "ID", Width: uiconst.ColWidthUUID}, {Title: "Name", Width: uiconst.ColWidthName}, {Title: "Status", Width: uiconst.ColWidthStatus}}
		rows := []table.Row{}
		for _, img := range imgList {
			rows = append(rows, table.Row{img.ID, img.Name, img.Status})
		}
		t := table.New(
			table.WithColumns(cols),
			table.WithRows(rows),
			table.WithFocused(true),
			table.WithHeight(m.height-uiconst.TableHeightOffset),
		)
		t.SetStyles(table.DefaultStyles())
		return imagesDataLoadedMsg{tbl: t, rows: rows}
	}
}

// Update handles messages for the model.
func (m ImagesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case imagesDataLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.table = msg.tbl
		m.allRows = msg.rows
		// Adjust columns and height based on current dimensions.
		m.updateTableColumns()
		m.table.SetHeight(m.height - 6)
		return m, nil
	case tea.WindowSizeMsg:
		// Update stored dimensions and adjust table.
		m.width = msg.Width
		m.height = msg.Height
		if m.table.Columns() != nil {
			m.table.SetHeight(m.height - 6)
			m.updateTableColumns()
		}
		return m, nil
	case tea.KeyMsg:
		if m.loading || m.err != nil {
			return m, nil
		}
		// Filter mode handling
		if !m.filterMode && msg.String() == "/" {
			m.filterMode = true
			m.filter.Focus()
			return m, textinput.Blink
		}
		if m.filterMode && msg.String() == "esc" {
			m.filterMode = false
			m.filter.Blur()
			m.filter.SetValue("")
			m.table.SetRows(m.allRows)
			return m, nil
		}
		if m.filterMode {
			var cmd tea.Cmd
			m.filter, cmd = m.filter.Update(msg)
			filterVal := m.filter.Value()
			if filterVal == "" {
				m.table.SetRows(m.allRows)
			} else {
				lower := strings.ToLower(filterVal)
				filtered := []table.Row{}
				for _, r := range m.allRows {
					for _, c := range r {
						if strings.Contains(strings.ToLower(c), lower) {
							filtered = append(filtered, r)
							break
						}
					}
				}
				m.table.SetRows(filtered)
			}
			return m, cmd
		}
		// Normal table navigation
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		return m, cmd
	default:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

// View renders the appropriate UI based on state.
func (m ImagesModel) View() string {
	if m.loading {
		return m.spinner.View()
	}
	if m.err != nil {
		return fmt.Sprintf("Error: %s", m.err)
	}
	if m.filterMode {
		filterLine := fmt.Sprintf("Filter: %s", m.filter.View())
		footer := "esc: clear"
		return fmt.Sprintf("%s\n%s\n%s", filterLine, m.table.View(), footer)
	}
	return m.table.View()
}

// updateTableColumns adjusts column widths based on the current width.
func (m *ImagesModel) updateTableColumns() {
	idW := uiconst.ColWidthUUID
	statusW := uiconst.ColWidthStatus
	// Compute flexible name width.
	nameW := m.width - idW - statusW - uiconst.TableHeightOffset
	if nameW < 10 {
		nameW = 10
	}
	m.table.SetColumns([]table.Column{{Title: "ID", Width: idW}, {Title: "Name", Width: nameW}, {Title: "Status", Width: statusW}})
}

// Table returns the underlying table model.
func (m ImagesModel) Table() table.Model { return m.table }

var _ tea.Model = (*ImagesModel)(nil)

// ImageDetailModel displays detailed information for a single image.
type ImageDetailModel struct {
	table   table.Model
	loading bool
	err     error
	spinner spinner.Model
	client  client.ImageClient
	imageID string
}

type imageDetailDataLoadedMsg struct {
	tbl table.Model
	err error
}

// NewImageDetailModel creates a new ImageDetailModel for the given image ID.
func NewImageDetailModel(ic client.ImageClient, imageID string) ImageDetailModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return ImageDetailModel{client: ic, loading: true, spinner: s, imageID: imageID}
}

// Init starts async loading of image details.
func (m ImageDetailModel) Init() tea.Cmd {
	return func() tea.Msg {
		img, err := m.client.GetImage(context.Background(), m.imageID)
		if err != nil {
			return imageDetailDataLoadedMsg{err: err}
		}
		cols := []table.Column{{Title: "Field", Width: uiconst.ColWidthField}, {Title: "Value", Width: uiconst.ColWidthValue}}
		rows := []table.Row{{"ID", img.ID}, {"Name", img.Name}, {"Status", img.Status}, {"MinDisk (GB)", fmt.Sprintf("%d", img.MinDisk)}, {"MinRAM (MB)", fmt.Sprintf("%d", img.MinRAM)}, {"Created", img.Created}, {"Updated", img.Updated}}
		t := table.New(
			table.WithColumns(cols),
			table.WithRows(rows),
			table.WithFocused(true),
		)
		t.SetStyles(table.DefaultStyles())
		return imageDetailDataLoadedMsg{tbl: t}
	}
}

// Update handles messages.
func (m ImageDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case imageDetailDataLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.table = msg.tbl
		return m, nil
	case tea.WindowSizeMsg:
		// Adjust table width to fill the terminal width.
		if !m.loading && len(m.table.Columns()) > 0 {
			cols := m.table.Columns()
			if len(cols) > 0 {
				// Compute a column width that distributes the available space evenly.
				// Subtract a small margin (4 characters) to avoid overflow.
				totalWidth := msg.Width - 4
				if totalWidth < 0 {
					totalWidth = msg.Width
				}
				colWidth := totalWidth / len(cols)
				if colWidth < 5 {
					colWidth = 5
				}
				for i := range cols {
					cols[i].Width = colWidth
				}
				m.table.SetColumns(cols)
				m.table.SetWidth(msg.Width)
			}
		}
		return m, nil
	case tea.KeyMsg:
		if m.loading || m.err != nil {
			return m, nil
		}
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		return m, cmd
	default:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

// View renders the image detail view.
func (m ImageDetailModel) View() string {
	if m.loading {
		return m.spinner.View()
	}
	if m.err != nil {
		return fmt.Sprintf("Error: %s", m.err)
	}
	return fmt.Sprintf("%s\n[esc] back", m.table.View())
}

// Table returns the underlying table model.
func (m ImageDetailModel) Table() table.Model { return m.table }

var _ tea.Model = (*ImageDetailModel)(nil)

// DeleteImage deletes an image by ID using the provided ImageClient.
func DeleteImage(ic client.ImageClient, imageID string) string {
	err := ic.DeleteImage(context.Background(), imageID)
	if err != nil {
		return fmt.Sprintf("Failed to delete image %s: %s", imageID, err)
	}
	return fmt.Sprintf("Image %s deleted successfully.", imageID)
}
