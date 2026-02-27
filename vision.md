# OpenStack TUI — Piano di Sviluppo

> TUI in Go con Bubble Tea per interrogare e gestire OpenStack.
> Utente target: sysadmin/devops tecnico. Auth via clouds.yaml.
> Scope: Compute (Nova), Network (Neutron), Storage (Cinder/Swift), Identity (Keystone).

---

## UX Vision

```
┌─────────────────────────────────────────────────────────────────┐
│ ☁  OpenStack TUI          cloud: homelab    project: admin  [?] │
├──────────────┬──────────────────────────────────────────────────┤
│ COMPUTE    ▶ │  Instances                              [r]efresh │
│ NETWORK      │ ┌──────┬────────────┬─────────┬────────┬───────┐ │
│ STORAGE      │ │ Name │ Status     │ Flavor  │ IP     │ Age   │ │
│ IDENTITY     │ ├──────┼────────────┼─────────┼────────┼───────┤ │
│              │ │ web1 │ ● ACTIVE   │ m1.small│ 10.0.1 │ 3d    │ │
│ [c]loud      │ │ db1  │ ● ACTIVE   │ m1.large│ 10.0.2 │ 12d   │ │
│ [p]roject    │ │ test │ ○ SHUTOFF  │ m1.tiny │ 10.0.3 │ 1h    │ │
│              │ └──────┴────────────┴─────────┴────────┴───────┘ │
│              │  [n]ew  [d]elete  [s]tart  [S]top  [enter] detail│
└──────────────┴──────────────────────────────────────────────────┘
```

**Principi UX per utente tecnico:**
- Navigazione keyboard-first (vim-like: `j/k` per muoversi, `enter` per drill-down)
- Nessun wizard — azioni dirette con form minimali
- Shortcut visibili sempre in footer
- Output raw disponibile con `y` (dump JSON della risorsa)
- Multi-cloud: switch cloud con `c`, switch project con `p`

---

## Struttura Repo

```
ostui/
├── cmd/
│   └── ostui/
│       └── main.go
├── internal/
│   ├── config/
│   │   ├── clouds.go          ← parsing clouds.yaml
│   │   └── clouds_test.go
│   ├── client/
│   │   ├── client.go          ← gophercloud client factory
│   │   ├── compute.go         ← Nova API wrapper
│   │   ├── network.go         ← Neutron API wrapper
│   │   ├── storage.go         ← Cinder/Swift API wrapper
│   │   ├── identity.go        ← Keystone API wrapper
│   │   └── client_test.go
│   ├── ui/
│   │   ├── app.go             ← root model Bubble Tea
│   │   ├── layout.go          ← sidebar + main panel layout
│   │   ├── keymap.go          ← keybindings centralizzate
│   │   ├── styles.go          ← lipgloss styles
│   │   ├── compute/
│   │   │   ├── instances.go   ← lista istanze
│   │   │   ├── detail.go      ← dettaglio istanza
│   │   │   ├── flavors.go
│   │   │   └── keypairs.go
│   │   ├── network/
│   │   │   ├── networks.go
│   │   │   ├── subnets.go
│   │   │   ├── floatingips.go
│   │   │   └── secgroups.go
│   │   ├── storage/
│   │   │   ├── volumes.go
│   │   │   ├── snapshots.go
│   │   │   └── buckets.go
│   │   ├── identity/
│   │   │   ├── projects.go
│   │   │   ├── users.go
│   │   │   └── tokens.go
│   │   └── common/
│   │       ├── table.go       ← componente tabella riusabile
│   │       ├── form.go        ← componente form riusabile
│   │       ├── confirm.go     ← dialog conferma azioni distruttive
│   │       ├── detail.go      ← pannello dettaglio key-value
│   │       └── statusbar.go
│   └── cache/
│       └── cache.go           ← cache TTL per ridurre API call
├── .opencode/
│   └── todo.md
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## Dipendenze

```go
require (
    github.com/charmbracelet/bubbletea       v1.x    // TUI framework
    github.com/charmbracelet/bubbles         v0.20.x // componenti (table, textinput, list)
    github.com/charmbracelet/lipgloss        v1.x    // styling
    github.com/gophercloud/gophercloud       v1.x    // OpenStack client
    github.com/gophercloud/utils             v0.x    // clouds.yaml parser
    gopkg.in/yaml.v3                                 // YAML parsing
)
```

**Nota:** usare `gophercloud/utils` per il parsing di `clouds.yaml` — gestisce già
`OS_CLOUD`, `OS_CLIENT_CONFIG_FILE`, merge di `clouds.yaml` + `secure.yaml`.

---

## Fase 0 — Scaffolding + Auth

### Obiettivo
Progetto Go compilabile, parsing `clouds.yaml`, client gophercloud funzionante,
selezione cloud da CLI e da TUI.

### clouds.yaml support
```go
// Lookup order (come openstack client):
// 1. $OS_CLIENT_CONFIG_FILE
// 2. ./clouds.yaml
// 3. ~/.config/openstack/clouds.yaml
// 4. /etc/openstack/clouds.yaml
//
// Cloud selection:
// 1. flag --cloud <name>
// 2. env $OS_CLOUD
// 3. primo cloud disponibile
// 4. selezione interattiva TUI se multipli
```

### CLI flags
```
ostui [flags]
  --cloud    <name>      cloud da usare (default: $OS_CLOUD)
  --project  <name>      project override
  --debug                stampa API calls su stderr
  --version              versione
```

### Prompt Fase 0
```bash
/opsx:new ostui
/opsx:ff

ralph "Inizializza modulo Go ostui.
Implementa internal/config/clouds.go che:
- Cerca clouds.yaml nel lookup order standard openstack
- Parsa il file con gophercloud/utils
- Espone CloudConfig struct con Auth, RegionName, Interface
- Supporta selezione cloud via --cloud flag o OS_CLOUD env

Implementa cmd/ostui/main.go con cobra, flag --cloud e --debug.
Scrivi clouds_test.go con fixture clouds.yaml di test.
go build ./... e go test ./... verdi.
Output <promise>COMPLETE</promise>." --max-iterations 10
```

---

## Fase 1 — Client Layer (gophercloud wrappers)

### Obiettivo
Wrapper tipizzati per le 4 API. Interfacce mockabili per i test.
Cache TTL per non hammering le API ad ogni render.

### Interfacce
```go
// internal/client/client.go
type ComputeClient interface {
    ListInstances(ctx context.Context) ([]Instance, error)
    GetInstance(ctx context.Context, id string) (*Instance, error)
    StartInstance(ctx context.Context, id string) error
    StopInstance(ctx context.Context, id string) error
    DeleteInstance(ctx context.Context, id string) error
    ListFlavors(ctx context.Context) ([]Flavor, error)
    ListKeypairs(ctx context.Context) ([]Keypair, error)
}

type NetworkClient interface {
    ListNetworks(ctx context.Context) ([]Network, error)
    ListSubnets(ctx context.Context) ([]Subnet, error)
    ListFloatingIPs(ctx context.Context) ([]FloatingIP, error)
    AllocateFloatingIP(ctx context.Context, networkID string) (*FloatingIP, error)
    ReleaseFloatingIP(ctx context.Context, id string) error
    ListSecurityGroups(ctx context.Context) ([]SecurityGroup, error)
}

type StorageClient interface {
    ListVolumes(ctx context.Context) ([]Volume, error)
    GetVolume(ctx context.Context, id string) (*Volume, error)
    DeleteVolume(ctx context.Context, id string) error
    ListSnapshots(ctx context.Context) ([]Snapshot, error)
    CreateSnapshot(ctx context.Context, volumeID, name string) (*Snapshot, error)
}

type IdentityClient interface {
    ListProjects(ctx context.Context) ([]Project, error)
    GetCurrentProject(ctx context.Context) (*Project, error)
    ListUsers(ctx context.Context) ([]User, error)
    GetTokenInfo(ctx context.Context) (*TokenInfo, error)
}
```

### Cache
```go
// TTL per tipo di risorsa (cambiano a velocità diverse)
const (
    TTLInstances     = 15 * time.Second  // cambiano spesso
    TTLNetworks      = 60 * time.Second  // stabili
    TTLFlavors       = 300 * time.Second // quasi statici
    TTLProjects      = 300 * time.Second // quasi statici
)
```

### Prompt Fase 1
```bash
# Prima fetch docs gophercloud via context7
ralph "Implementa internal/client/ con wrapper gophercloud per:
- compute.go: ListInstances, GetInstance, Start/Stop/DeleteInstance, ListFlavors, ListKeypairs
- network.go: ListNetworks, ListSubnets, ListFloatingIPs, AllocateFloatingIP, ReleaseFloatingIP, ListSecurityGroups
- storage.go: ListVolumes, GetVolume, DeleteVolume, ListSnapshots, CreateSnapshot
- identity.go: ListProjects, GetCurrentProject, ListUsers, GetTokenInfo

Usa interfacce mockabili. Implementa cache TTL in cache/cache.go.
Scrivi test con mock (non colpire OpenStack reale).
go test ./internal/client/... verde.
Output <promise>COMPLETE</promise>." --max-iterations 20
```

---

## Fase 2 — TUI Foundation

### Obiettivo
Layout base funzionante: sidebar navigabile, panel principale,
statusbar, switch cloud/project, keybindings.

### Architettura Bubble Tea
```
AppModel (root)
├── state: sidebar focus | main focus | modal
├── activeSection: compute | network | storage | identity
├── SidebarModel
│   └── lista sezioni navigabile con j/k
├── MainModel (switcha in base a activeSection)
│   ├── ComputeModel
│   ├── NetworkModel
│   ├── StorageModel
│   └── IdentityModel
├── StatusBarModel
│   └── cloud, project, loading indicator, errori
└── ModalModel (opzionale, per confirm/form)
```

### Keybindings globali
```
j/k / ↑↓      navigare lista
enter          drill-down / conferma
esc            torna indietro / chiudi modal
tab            switch sidebar ↔ main panel
c              switch cloud
p              switch project
r              refresh (forza invalidazione cache)
y              dump JSON raw della risorsa selezionata
?              help overlay
q / ctrl+c     quit
```

### Keybindings per sezione
```
COMPUTE
  n   new instance (form)
  d   delete instance (confirm dialog)
  s   start instance
  S   stop instance
  l   view logs (console output)

NETWORK
  n   create network
  f   allocate floating IP
  F   release floating IP
  d   delete risorsa selezionata

STORAGE
  n   create volume
  s   create snapshot del volume selezionato
  d   delete

IDENTITY
  (read-only per sicurezza — nessuna azione distruttiva)
```

### Stile (lipgloss)
```go
// Palette ispirata a colori OpenStack ufficiali
var (
    ColorPrimary   = lipgloss.Color("#D13C3C")  // rosso OpenStack
    ColorSecondary = lipgloss.Color("#4A90D9")  // blu accento
    ColorSuccess   = lipgloss.Color("#5CB85C")  // verde ACTIVE
    ColorWarning   = lipgloss.Color("#F0AD4E")  // arancione WARNING
    ColorMuted     = lipgloss.Color("#666666")  // grigio testo secondario
    ColorBorder    = lipgloss.Color("#333333")  // bordi
)

// Status istanza → colore
var StatusColors = map[string]lipgloss.Color{
    "ACTIVE":   ColorSuccess,
    "SHUTOFF":  ColorMuted,
    "ERROR":    ColorPrimary,
    "BUILD":    ColorWarning,
    "DELETING": ColorWarning,
}
```

### Prompt Fase 2
```bash
ralph "Implementa la TUI foundation in internal/ui/:
- app.go: AppModel root con state machine (sidebar/main/modal)
- layout.go: layout sidebar 20% + main 80% con lipgloss
- keymap.go: keybindings centralizzate con charmbracelet/bubbles/key
- styles.go: palette colori OpenStack, stili bordi/header/statusbar
- common/table.go: componente tabella riusabile (wrap bubbles/table)
- common/statusbar.go: cloud, project, loading spinner, ultimo errore

La app deve avviarsi, mostrare il layout, navigare sidebar con j/k,
switch sezioni con tab. Dati placeholder (no API reali ancora).
go build ./... verde, ostui si avvia senza crash.
Output <promise>COMPLETE</promise>." --max-iterations 20
```

---

## Fase 3 — Sezione Compute

### Obiettivo
Prima sezione completamente funzionante end-to-end con API reali.

### Viste
**Lista Istanze:**
```
Name     Status    Flavor     IP           AZ      Age
──────────────────────────────────────────────────────
web1     ● ACTIVE  m1.small   10.0.1.10    nova    3d
db1      ● ACTIVE  m1.large   10.0.1.11    nova    12d
test     ○ SHUTOFF m1.tiny    10.0.1.12    nova    1h
```

**Dettaglio Istanza (enter):**
```
Instance: web1
──────────────────────────────
ID          8fa3c2...
Status      ACTIVE
Flavor      m1.small (1 vCPU, 2GB RAM, 20GB disk)
Image       Ubuntu 22.04
IP (fixed)  10.0.1.10
IP (float)  203.0.113.5
Key Pair    mykey
AZ          nova
Created     2024-01-10 14:32:00
Host        compute01

[s]tart  [S]top  [d]elete  [l]ogs  [y]json  [esc]back
```

**Form Nuova Istanza (n):**
```
New Instance
────────────────────
Name        [____________]
Image       [Ubuntu 22.04          ▼]
Flavor      [m1.small              ▼]
Network     [internal              ▼]
Key Pair    [mykey                 ▼]
────────────────────
[enter] Create   [esc] Cancel
```

### Prompt Fase 3
```bash
ralph "Implementa internal/ui/compute/ con viste Instances, Detail, Flavors, Keypairs.
Integra con ComputeClient reale (via interface, mockabile).
Lista istanze con tabella navigabile j/k, enter per dettaglio.
Dettaglio mostra tutti i campi, keybinding s/S/d/l/y.
Form nuova istanza con select per image/flavor/network/keypair.
Dialog conferma per delete (azione distruttiva).
Caricamento asincrono (spinner mentre chiama API, non blocca UI).
go build ./... verde.
Output <promise>COMPLETE</promise>." --max-iterations 25
```

---

## Fase 4 — Sezione Network

### Viste principali
**Networks:**
```
Name        Status  Shared  External  Subnets
─────────────────────────────────────────────
internal    UP      No      No        192.168.1.0/24
external    UP      Yes     Yes       203.0.113.0/24
```

**Floating IPs:**
```
IP              Status      Associated To    Pool
───────────────────────────────────────────────────
203.0.113.5     ACTIVE      web1 (10.0.1.10) external
203.0.113.8     DOWN        —                external

[f]allocate  [F]release  [a]associate  [A]disassociate
```

**Security Groups:**
```
Name      Description         Rules
────────────────────────────────────
default   Default sec group   3 rules
web       Web traffic         4 rules
```

**Dettaglio Security Group (enter):**
```
Rules for: web
Direction  Protocol  Port Range  Remote
───────────────────────────────────────────
ingress    tcp       80          0.0.0.0/0
ingress    tcp       443         0.0.0.0/0
ingress    tcp       22          10.0.0.0/8
egress     any       any         0.0.0.0/0
```

### Prompt Fase 4
```bash
ralph "Implementa internal/ui/network/ con viste Networks, Subnets, FloatingIPs, SecurityGroups.
FloatingIPs: allocate (seleziona pool), release (confirm), associate/disassociate da istanza.
SecurityGroups: lista regole in dettaglio, read-only per ora.
Caricamento asincrono. go build ./... verde.
Output <promise>COMPLETE</promise>." --max-iterations 20
```

---

## Fase 5 — Sezione Storage

### Viste principali
**Volumes:**
```
Name     Size   Status     Attached To   Type      Age
──────────────────────────────────────────────────────
data1    50GB   in-use     db1           standard  5d
backup   100GB  available  —             fast      2h
```

**Dettaglio Volume (enter):**
```
Volume: data1
──────────────────────────────
ID          a8f2d1...
Status      in-use
Size        50 GB
Type        standard
Attached    db1 (/dev/vdb)
Bootable    No
Created     2024-01-05 09:00:00

[s]napshot  [d]elete  [y]json  [esc]back
```

**Snapshots:**
```
Name          Volume    Size   Status     Age
─────────────────────────────────────────────
data1-snap1   data1     50GB   available  1d
data1-snap2   data1     50GB   available  5h
```

### Prompt Fase 5
```bash
ralph "Implementa internal/ui/storage/ con viste Volumes e Snapshots.
Volumes: lista, dettaglio, create snapshot (form con nome), delete (confirm).
Snapshots: lista con volume sorgente, delete (confirm).
Caricamento asincrono. go build ./... verde.
Output <promise>COMPLETE</promise>." --max-iterations 20
```

---

## Fase 6 — Sezione Identity

### Read-only per sicurezza. Nessuna azione distruttiva.

### Viste
**Projects:**
```
Name        ID          Domain    Status
─────────────────────────────────────────
admin       abc123      Default   enabled
demo        def456      Default   enabled
► myproject ghi789      Default   enabled   ← current
```

**Users:**
```
Name     Email                Domain    Status
───────────────────────────────────────────────
admin    admin@example.com    Default   enabled
user1    user1@example.com    Default   enabled
```

**Token Info:**
```
Current Token
──────────────────────────────
User        admin
Project     myproject
Issued      2024-01-15 10:00:00
Expires     2024-01-15 14:00:00  (3h 42m remaining)
Roles       admin, member
Endpoints   compute, network, volume, identity
```

### Prompt Fase 6
```bash
ralph "Implementa internal/ui/identity/ con viste Projects, Users, TokenInfo.
Tutte read-only — nessuna azione distruttiva.
Evidenzia il project corrente nella lista.
Token info mostra scadenza con countdown colorato (verde/giallo/rosso).
go build ./... verde.
Output <promise>COMPLETE</promise>." --max-iterations 15
```

---

## Fase 7 — Switch Cloud e Project

### Obiettivo
Selezionare cloud e project dinamicamente senza riavviare.

### Switch Cloud (`c`)
```
Select Cloud
────────────────
► homelab
  production
  staging

[j/k] navigate  [enter] select  [esc] cancel
```

→ ri-inizializza il client gophercloud con le nuove credenziali
→ invalida tutta la cache
→ aggiorna statusbar

### Switch Project (`p`)
```
Select Project
────────────────
► admin
  demo
  myproject

[j/k] navigate  [enter] select  [esc] cancel
```

→ cambia scope project sul token
→ invalida cache compute/network/storage (non identity)

### Prompt Fase 7
```bash
ralph "Implementa switch cloud (c) e switch project (p).
Switch cloud: legge tutti i cloud da clouds.yaml, mostra lista, ri-inizializza client.
Switch project: lista projects da Keystone, cambia scope, invalida cache.
Statusbar si aggiorna dopo ogni switch.
go build ./... verde.
Output <promise>COMPLETE</promise>." --max-iterations 15
```

---

## Fase 8 — Polish e Release

### Obiettivo
TUI pronta per uso quotidiano e pubblicabile.

### Checklist
- [ ] Help overlay (`?`) con tutti i keybinding per sezione attiva
- [ ] JSON dump (`y`) formattato con syntax highlight (lipgloss colors)
- [ ] Flag `--debug` scrive API calls su file di log (`~/.ostui/debug.log`)
- [ ] Gestione errori API uniforme (statusbar rossa + messaggio)
- [ ] Auto-refresh configurabile (`--refresh 30s`)
- [ ] README con GIF demo, installazione, configurazione
- [ ] Goreleaser per binary cross-platform (Linux/macOS/arm64)
- [ ] `go vet` e `golangci-lint` verdi

### Prompt Fase 8
```bash
ralph "Aggiungi:
1. Help overlay con ? che mostra keybinding contestuali per sezione attiva
2. JSON dump con y formattato e colorato
3. Gestione errori uniforme: qualsiasi errore API → statusbar rossa + messaggio
4. Flag --refresh <duration> per auto-refresh periodico
5. Makefile con target: build, test, lint, release
6. README.md con installazione e configurazione clouds.yaml

go build ./..., go test ./..., golangci-lint verdi.
Output <promise>COMPLETE</promise>." --max-iterations 20
```

---

## Sequenza Completa con il tuo Workflow

```bash
# Setup progetto
mkdir ~/projects/ostui && cd ~/projects/ostui
/opsx:new ostui
/opsx:ff   # rivedi proposal.md e tasks.md

# Fetch docs gophercloud prima di iniziare
# context7 → resolve "gophercloud" → /gophercloud/gophercloud
# context7 → resolve "bubbletea" → /charmbracelet/bubbletea

# Fasi in sequenza con ralph
ralph "$(cat .opencode/prompts/phase0-scaffolding.md)" --max-iterations 10
ralph "$(cat .opencode/prompts/phase1-client.md)"      --max-iterations 20
ralph "$(cat .opencode/prompts/phase2-foundation.md)"  --max-iterations 20
ralph "$(cat .opencode/prompts/phase3-compute.md)"     --max-iterations 25
ralph "$(cat .opencode/prompts/phase4-network.md)"     --max-iterations 20
ralph "$(cat .opencode/prompts/phase5-storage.md)"     --max-iterations 20
ralph "$(cat .opencode/prompts/phase6-identity.md)"    --max-iterations 15
ralph "$(cat .opencode/prompts/phase7-switch.md)"      --max-iterations 15
ralph "$(cat .opencode/prompts/phase8-polish.md)"      --max-iterations 20

# Review finale
opencode review
```

---

## Rischi e Mitigazioni

| Rischio | Probabilità | Mitigazione |
|---------|------------|-------------|
| gophercloud API diverse per versione OpenStack | Media | Astrarre dietro interfacce — swap implementazione senza toccare UI |
| Bubble Tea layout complesso (sidebar + panel) | Media | Iniziare con layout fisso, lipgloss flex solo nella fase polish |
| Azioni distruttive senza conferma | Bassa | `common/confirm.go` obbligatorio per delete/release — gating in keymap |
| API lente bloccano UI | Media | Tutto asincrono via `tea.Cmd` — spinner sempre visibile durante loading |
| Token scaduto a runtime | Bassa | gophercloud gestisce rinnovo automatico — intercettare 401 e mostrare messaggio |
| Rate limiting API OpenStack | Bassa | Cache TTL riduce drasticamente le chiamate |

---

## Definition of Done per ogni Fase

- [ ] `go build ./...` verde
- [ ] `go test ./...` verde, coverage > 60% sul client layer
- [ ] Nessun `go vet` warning
- [ ] La feature funziona end-to-end contro un OpenStack reale (o DevStack)
- [ ] Nessuna azione distruttiva senza confirm dialog

---

## Milestone

```
v0.1  Fase 0-2  → TUI navigabile con dati placeholder
v0.2  Fase 3    → Compute completamente funzionante
v0.3  Fase 4-5  → Network + Storage
v0.4  Fase 6-7  → Identity + multi-cloud/project
v1.0  Fase 8    → Polish, README, release binaries
```

---

*Stack: Go + Bubble Tea + gophercloud | Auth: clouds.yaml | Target: sysadmin/devops*
