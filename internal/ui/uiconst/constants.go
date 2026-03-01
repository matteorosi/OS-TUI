package uiconst

// Table column widths
const (
	ColWidthUUID         = 36 // Standard OpenStack UUID
	ColWidthName         = 20 // Name columns
	ColWidthStatus       = 12 // Status/State columns
	ColWidthCIDR         = 20 // CIDR/IP columns
	ColWidthIPVersion    = 6  // IP version (4/6)
	ColWidthEnabled      = 8  // Boolean columns
	ColWidthDescription  = 30 // Description columns
	ColWidthStateful     = 8  // Stateful boolean
	ColWidthDirection    = 8  // Direction (ingress/egress)
	ColWidthEtherType    = 8  // EtherType
	ColWidthProtocol     = 6  // Protocol
	ColWidthPortRange    = 12 // Port range
	ColWidthRemoteIP     = 15 // Remote IP
	ColWidthFixed        = 15 // Fixed IP
	ColWidthError        = 80 // Error message column
	ColWidthField        = 20 // Field name in detail views
	ColWidthValue        = 60 // Value in detail views
	ColWidthValueShort   = 30 // Short value in two-column detail views
	ColWidthProvisioning = 14 // Provisioning status column width
	ColWidthOperating    = 12 // Operating status column width (same as status)
	ColWidthVIPAddress   = 16 // VIP address column width
	ColWidthAlgorithm    = 16 // Algorithm column width
	ColWidthNameLong     = 30 // Longer name column width (e.g., load balancer name)
	ColWidthFingerprint  = 30 // Fingerprint column width
	ColWidthType         = 10 // Type column width
	ColWidthTTL          = 8  // TTL column width
	ColWidthRecords      = 30 // Records column width
	ColWidthNameDNS      = 40 // DNS name column width
	ColWidthSize         = 8  // Size column width (e.g., volume size)
	ColWidthPort         = 6  // Port column width (same as protocol)
	ColWidthStatusLong   = 14 // Longer status column width (e.g., load balancer status)
	ColWidthRAMUsed      = 9  // RAM used column width
	ColWidthDiskUsed     = 9  // Disk used column width
)

// Table height constants
const (
	TableHeightOffset  = 6  // Subtracted from terminal height: m.height - TableHeightOffset
	TableHeightDefault = 20 // Default height for static tables (render helpers)
)
