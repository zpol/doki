package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	containersvc "github.com/zpol/doki/internal/app/containers"
	diagsvc "github.com/zpol/doki/internal/app/diagnostics"
	hostsvc "github.com/zpol/doki/internal/app/host"
	logsvc "github.com/zpol/doki/internal/app/logs"
	metricssvc "github.com/zpol/doki/internal/app/metrics"
	resourcessvc "github.com/zpol/doki/internal/app/resources"
	secsvc "github.com/zpol/doki/internal/app/security"
	shellsvc "github.com/zpol/doki/internal/app/shell"
	"github.com/zpol/doki/internal/config"
	"github.com/zpol/doki/internal/domain/models"
	"github.com/zpol/doki/internal/domain/ports"
	"github.com/zpol/doki/internal/tui/components"
	"github.com/zpol/doki/internal/tui/dialogs"
	"github.com/zpol/doki/internal/tui/keymap"
	"github.com/zpol/doki/internal/tui/screens/dashboard"
	containerscreen "github.com/zpol/doki/internal/tui/screens/containers"
	diagscreen "github.com/zpol/doki/internal/tui/screens/diagnostics"
	inspectscreen "github.com/zpol/doki/internal/tui/screens/inspect"
	logscreen "github.com/zpol/doki/internal/tui/screens/logs"
	composescreen "github.com/zpol/doki/internal/tui/screens/compose"
	detailscreen "github.com/zpol/doki/internal/tui/screens/detail"
	resscreen "github.com/zpol/doki/internal/tui/screens/resources"
	secscreen "github.com/zpol/doki/internal/tui/screens/security"
	"github.com/zpol/doki/internal/tui/themes"
)

type viewState int

const (
	viewDashboard viewState = iota
	viewContainers
	viewInspect
	viewDetail
	viewLogs
	viewImages
	viewVolumes
	viewNetworks
	viewCompose
	viewSecurity
	viewDiagnostics
	viewHelp
)

// Dependencies wires application services into the TUI.
type Dependencies struct {
	Containers *containersvc.Service
	Host       *hostsvc.Service
	Logs       *logsvc.Service
	Metrics    *metricssvc.Service
	Resources    *resourcessvc.Service
	Shell        *shellsvc.Service
	Security     *secsvc.Engine
	Diagnostics  *diagsvc.Engine
	Docker       ports.DockerClient
	Config       config.Config
}

// Model is the root Bubble Tea model for DOKI.
type Model struct {
	deps   Dependencies
	styles themes.Styles
	keys   keymap.Registry

	width  int
	height int
	view   viewState

	header   components.Header
	palette  components.Palette
	list     containerscreen.Screen
	inspect  inspectscreen.Screen
	detail   detailscreen.Screen
	logs     logscreen.Screen
	dash     dashboard.Screen
	images   resscreen.Screen
	volumes  resscreen.Screen
	networks resscreen.Screen
	compose  composescreen.Screen
	security secscreen.Screen
	diags    diagscreen.Screen
	confirm  dialogs.Confirm

	host       models.HostInfo
	containers []models.Container
	stats      []models.ContainerStats
	events     []models.Event
	filter     string
	errText    string
	statusMsg  string
	helpOpen   bool

	pendingAction string
	pendingID     string
	pendingName   string
	pendingBulk   []models.Container

	logCancel context.CancelFunc
	evtCancel context.CancelFunc
	eventCh   <-chan models.Event
	eventErr  <-chan error

	refreshEvery time.Duration
}

// New creates the root TUI model.
func New(deps Dependencies) Model {
	theme := themes.ByName(deps.Config.Theme)
	styles := themes.NewStyles(theme)
	refresh := deps.Config.Refresh.Containers
	if refresh < 500*time.Millisecond {
		refresh = time.Second
	}
	return Model{
		deps:         deps,
		styles:       styles,
		keys:         keymap.Default(),
		view:         viewContainers,
		header:       components.NewHeader(styles),
		palette:      components.NewPalette(styles),
		list:         containerscreen.New(styles),
		inspect:      inspectscreen.New(styles),
		detail:       detailscreen.New(styles),
		logs:         logscreen.New(styles),
		dash:         dashboard.New(styles),
		images:       resscreen.New(styles, resscreen.KindImages),
		volumes:      resscreen.New(styles, resscreen.KindVolumes),
		networks:     resscreen.New(styles, resscreen.KindNetworks),
		compose:      composescreen.New(styles),
		security:     secscreen.New(styles),
		diags:        diagscreen.New(styles),
		confirm:      dialogs.NewConfirm(styles),
		refreshEvery: refresh,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.refreshHostCmd(),
		m.refreshContainersCmd(),
		m.startEventsCmd(),
		tickCmd(m.refreshEvery),
	)
}

type tickMsg time.Time
type hostMsg struct {
	info models.HostInfo
	err  error
}
type containersMsg struct {
	items []models.Container
	err   error
}
type inspectMsg struct {
	detail models.ContainerDetails
	err    error
}
type imageDetailMsg struct {
	detail models.ImageDetails
	err    error
}
type volumeDetailMsg struct {
	detail models.VolumeDetails
	err    error
}
type networkDetailMsg struct {
	detail models.NetworkDetails
	err    error
}
type actionDoneMsg struct {
	action string
	name   string
	err    error
	ok     int
	fail   int
	bulk   bool
}
type statsMsg struct {
	stats []models.ContainerStats
}
type inspectStatsMsg struct {
	stats models.ContainerStats
}
type eventMsg struct {
	event models.Event
}
type eventsErrMsg struct{ err error }
type logLineMsg struct {
	line  models.LogLine
	lines <-chan models.LogLine
	errs  <-chan error
}
type logErrMsg struct{ err error }
type imagesMsg struct {
	items []models.Image
	err   error
}
type volumesMsg struct {
	items []models.Volume
	err   error
}
type networksMsg struct {
	items []models.Network
	err   error
}
type securityMsg struct {
	items []models.Finding
	err   error
}
type diagnosticsMsg struct {
	items []models.DiagnosticFinding
	err   error
}
type shellDoneMsg struct{ err error }

func tickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m Model) refreshHostCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		info, err := m.deps.Host.Info(ctx)
		return hostMsg{info: info, err: err}
	}
}

func (m Model) refreshContainersCmd() tea.Cmd {
	query := m.filter
	showStopped := m.deps.Config.Display.ShowStopped
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		items, err := m.deps.Containers.List(ctx, containersvc.ListOptions{
			ShowStopped: showStopped,
			Query:       query,
		})
		return containersMsg{items: items, err: err}
	}
}

func (m Model) refreshStatsCmd(items []models.Container) tea.Cmd {
	ids := make([]string, 0, 12)
	for _, c := range items {
		if c.IsRunning() {
			ids = append(ids, c.ID)
		}
		if len(ids) >= 12 {
			break
		}
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		stats, _ := m.deps.Metrics.Sample(ctx, ids)
		return statsMsg{stats: stats}
	}
}

func (m Model) refreshInspectStatsCmd(id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		stats, err := m.deps.Metrics.SampleOne(ctx, []string{id})
		if err != nil || len(stats) == 0 {
			return inspectStatsMsg{stats: models.ContainerStats{ContainerID: id}}
		}
		return inspectStatsMsg{stats: stats[0]}
	}
}

func (m Model) refreshImagesCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		items, err := m.deps.Resources.Images(ctx, "")
		return imagesMsg{items: items, err: err}
	}
}

func (m Model) refreshVolumesCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		items, err := m.deps.Resources.Volumes(ctx, "")
		return volumesMsg{items: items, err: err}
	}
}

func (m Model) refreshNetworksCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		items, err := m.deps.Resources.Networks(ctx, "")
		return networksMsg{items: items, err: err}
	}
}

func (m Model) refreshSecurityCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		items, err := m.deps.Security.InspectAll(ctx)
		return securityMsg{items: items, err: err}
	}
}

func (m Model) refreshDiagnosticsCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		items, err := m.deps.Diagnostics.AnalyzeAll(ctx)
		return diagnosticsMsg{items: items, err: err}
	}
}

func (m Model) inspectCmd(id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		detail, err := m.deps.Containers.Inspect(ctx, id)
		return inspectMsg{detail: detail, err: err}
	}
}

func (m Model) inspectImageCmd(id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		detail, err := m.deps.Resources.InspectImage(ctx, id)
		return imageDetailMsg{detail: detail, err: err}
	}
}

func (m Model) inspectVolumeCmd(name string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		detail, err := m.deps.Resources.InspectVolume(ctx, name)
		return volumeDetailMsg{detail: detail, err: err}
	}
}

func (m Model) inspectNetworkCmd(id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		detail, err := m.deps.Resources.InspectNetwork(ctx, id)
		return networkDetailMsg{detail: detail, err: err}
	}
}

func (m Model) runActionCmd(action, id, name string) tea.Cmd {
	return m.runBulkActionCmd(action, []models.Container{{ID: id, Name: name}})
}

func (m Model) runBulkActionCmd(action string, targets []models.Container) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(30*len(targets)+30)*time.Second)
		defer cancel()
		ok, fail := 0, 0
		var lastErr error
		names := make([]string, 0, len(targets))
		for _, c := range targets {
			names = append(names, c.Name)
			var err error
			switch action {
			case "start":
				err = m.deps.Containers.Start(ctx, c.ID)
			case "stop":
				err = m.deps.Containers.Stop(ctx, c.ID)
			case "restart":
				err = m.deps.Containers.Restart(ctx, c.ID)
			case "pause":
				err = m.deps.Containers.Pause(ctx, c.ID)
			case "unpause":
				err = m.deps.Containers.Unpause(ctx, c.ID)
			case "remove":
				err = m.deps.Containers.Remove(ctx, c.ID, true)
			default:
				err = fmt.Errorf("unknown action %q", action)
			}
			if err != nil {
				fail++
				lastErr = err
				continue
			}
			ok++
		}
		label := names[0]
		if len(names) > 1 {
			label = fmt.Sprintf("%d containers", len(names))
		}
		return actionDoneMsg{
			action: action,
			name:   label,
			err:    lastErr,
			ok:     ok,
			fail:   fail,
			bulk:   len(targets) > 1,
		}
	}
}

func (m Model) startEventsCmd() tea.Cmd {
	return func() tea.Msg {
		return eventBootstrapMsg{}
	}
}

type eventBootstrapMsg struct{}

func waitEventCmd(events <-chan models.Event, errs <-chan error) tea.Cmd {
	return func() tea.Msg {
		select {
		case ev, ok := <-events:
			if !ok {
				return nil
			}
			return eventMsg{event: ev}
		case err, ok := <-errs:
			if !ok || err == nil {
				return nil
			}
			return eventsErrMsg{err: err}
		}
	}
}

func (m Model) openLogsCmd(id, name string) tea.Cmd {
	return func() tea.Msg {
		return openLogsMsg{id: id, name: name}
	}
}

type openLogsMsg struct {
	id   string
	name string
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m = m.layout()
		return m, nil

	case tickMsg:
		cmds := []tea.Cmd{tickCmd(m.refreshEvery)}
		if !m.confirm.Active && !m.palette.Active() && !m.list.Filtering() && m.view != viewLogs {
			cmds = append(cmds, m.refreshHostCmd(), m.refreshContainersCmd())
			if m.view == viewInspect {
				if id := m.inspect.ContainerID(); id != "" {
					cmds = append(cmds, m.refreshInspectStatsCmd(id))
				}
			}
		}
		return m, tea.Batch(cmds...)

	case hostMsg:
		if msg.err != nil {
			m.errText = friendlyDockerError(msg.err)
			msg.info.State = models.ConnectionError
			msg.info.LastError = msg.err.Error()
		} else {
			m.errText = ""
		}
		m.host = msg.info
		m.header = m.header.SetHost(msg.info).SetError(m.errText)
		m.dash = m.dash.SetHost(msg.info)
		return m, nil

	case containersMsg:
		if msg.err != nil {
			m.errText = friendlyDockerError(msg.err)
			m.header = m.header.SetError(m.errText)
			m.list = m.list.SetStatus("docker unavailable")
			return m, nil
		}
		m.containers = msg.items
		m.list = m.list.SetItems(msg.items).SetStatus(m.statusMsg)
		m.dash = m.dash.SetContainers(msg.items)
		m.compose = m.compose.SetContainers(msg.items)
		return m, m.refreshStatsCmd(msg.items)

	case statsMsg:
		m.stats = msg.stats
		cpu := map[string][]float64{}
		mem := map[string][]float64{}
		for _, st := range msg.stats {
			c, mm := m.deps.Metrics.History(st.ContainerID)
			cpu[st.ContainerID] = c
			mem[st.ContainerID] = mm
		}
		aggCPU, aggMem, netRx, netTx, blkR, blkW := m.deps.Metrics.AggregateHistory()
		m.dash = m.dash.
			SetStats(msg.stats, cpu, mem).
			SetAggregate(m.deps.Metrics.Aggregate(), aggCPU, aggMem, netRx, netTx, blkR, blkW)
		return m, nil

	case inspectStatsMsg:
		cpu, mem := m.deps.Metrics.History(msg.stats.ContainerID)
		m.inspect = m.inspect.SetStats(msg.stats, cpu, mem)
		return m, nil

	case eventBootstrapMsg:
		if m.evtCancel != nil {
			m.evtCancel()
		}
		ctx, cancel := context.WithCancel(context.Background())
		m.evtCancel = cancel
		m.eventCh, m.eventErr = m.deps.Docker.Events(ctx)
		return m, waitEventCmd(m.eventCh, m.eventErr)

	case eventMsg:
		m.events = append(m.events, msg.event)
		if len(m.events) > 100 {
			m.events = m.events[len(m.events)-100:]
		}
		m.dash = m.dash.SetEvents(m.events)
		return m, waitEventCmd(m.eventCh, m.eventErr)

	case eventsErrMsg:
		m.statusMsg = "events: " + msg.err.Error()
		return m, nil

	case openLogsMsg:
		if m.logCancel != nil {
			m.logCancel()
		}
		m.view = viewLogs
		m.header = m.header.SetView("LOGS")
		m.logs = m.logs.SetTarget(msg.name)
		ctx, cancel := context.WithCancel(context.Background())
		m.logCancel = cancel
		lines, errs, _, err := m.deps.Logs.Stream(ctx, msg.id, ports.LogOptions{
			Follow:     true,
			Tail:       "300",
			Timestamps: true,
			Stdout:     true,
			Stderr:     true,
		})
		if err != nil {
			m.logs = m.logs.SetStatus(err.Error())
			return m, nil
		}
		return m, waitLogCmd(lines, errs)

	case logLineMsg:
		m.logs = m.logs.Append(msg.line)
		return m, waitLogCmd(msg.lines, msg.errs)

	case logErrMsg:
		m.logs = m.logs.SetStatus(msg.err.Error())
		return m, nil

	case imagesMsg:
		if msg.err != nil {
			m.statusMsg = msg.err.Error()
			return m, nil
		}
		m.images = m.images.SetImages(msg.items)
		return m, nil
	case volumesMsg:
		if msg.err != nil {
			m.statusMsg = msg.err.Error()
			return m, nil
		}
		m.volumes = m.volumes.SetVolumes(msg.items)
		return m, nil
	case networksMsg:
		if msg.err != nil {
			m.statusMsg = msg.err.Error()
			return m, nil
		}
		m.networks = m.networks.SetNetworks(msg.items)
		return m, nil
	case securityMsg:
		if msg.err != nil {
			m.security = m.security.SetStatus(msg.err.Error())
			return m, nil
		}
		m.security = m.security.SetFindings(msg.items).SetStatus(fmt.Sprintf("%d findings", len(msg.items)))
		return m, nil
	case diagnosticsMsg:
		if msg.err != nil {
			m.diags = m.diags.SetStatus(msg.err.Error())
			return m, nil
		}
		m.diags = m.diags.SetFindings(msg.items).SetStatus(fmt.Sprintf("%d findings", len(msg.items)))
		return m, nil

	case inspectMsg:
		if msg.err != nil {
			m.inspect = m.inspect.SetError(msg.err.Error())
		} else {
			m.inspect = m.inspect.SetDetail(msg.detail)
		}
		return m, nil

	case imageDetailMsg:
		if msg.err != nil {
			m.detail = m.detail.SetError(msg.err.Error())
		} else {
			m.detail = m.detail.SetImage(msg.detail)
		}
		return m, nil
	case volumeDetailMsg:
		if msg.err != nil {
			m.detail = m.detail.SetError(msg.err.Error())
		} else {
			m.detail = m.detail.SetVolume(msg.detail)
		}
		return m, nil
	case networkDetailMsg:
		if msg.err != nil {
			m.detail = m.detail.SetError(msg.err.Error())
		} else {
			m.detail = m.detail.SetNetwork(msg.detail)
		}
		return m, nil

	case actionDoneMsg:
		switch {
		case msg.bulk:
			m.statusMsg = fmt.Sprintf("%s: %d ok, %d failed", strings.ToUpper(msg.action), msg.ok, msg.fail)
			if msg.fail == 0 {
				m.list = m.list.ClearMarks()
			}
		case msg.err != nil:
			m.statusMsg = fmt.Sprintf("%s failed: %v", msg.action, msg.err)
		default:
			m.statusMsg = fmt.Sprintf("%s %s", strings.ToUpper(msg.action), msg.name)
			m.list = m.list.ClearMarks()
		}
		m.list = m.list.SetStatus(m.statusMsg)
		return m, m.refreshContainersCmd()

	case shellDoneMsg:
		if msg.err != nil {
			m.statusMsg = "shell: " + msg.err.Error()
			m.list = m.list.SetStatus(m.statusMsg)
		}
		return m, m.refreshContainersCmd()

	case containerscreen.FilterChangedMsg:
		m.filter = msg.Query
		return m, m.refreshContainersCmd()

	case components.PaletteCommand:
		return m.handlePalette(msg)

	case dialogs.ConfirmResult:
		if !msg.Confirmed {
			m.pendingAction, m.pendingID, m.pendingName = "", "", ""
			m.pendingBulk = nil
			return m, nil
		}
		action, id, name, bulk := m.pendingAction, m.pendingID, m.pendingName, m.pendingBulk
		m.pendingAction, m.pendingID, m.pendingName = "", "", ""
		m.pendingBulk = nil
		if len(bulk) > 0 {
			return m, m.runBulkActionCmd(action, bulk)
		}
		return m, m.runActionCmd(action, id, name)

	case tea.KeyPressMsg:
		if m.confirm.Active {
			var cmd tea.Cmd
			m.confirm, cmd = m.confirm.Update(msg)
			return m, cmd
		}
		if m.palette.Active() {
			var cmd tea.Cmd
			m.palette, cmd = m.palette.Update(msg)
			return m, cmd
		}
		if m.helpOpen || m.view == viewHelp {
			switch msg.String() {
			case "esc", "?", "q":
				m.helpOpen = false
				m.view = viewContainers
				m.header = m.header.SetView("CONTAINERS")
			}
			return m, nil
		}
		return m.handleKey(msg)
	}
	return m, nil
}

func waitLogCmd(lines <-chan models.LogLine, errs <-chan error) tea.Cmd {
	return func() tea.Msg {
		select {
		case line, ok := <-lines:
			if !ok {
				return nil
			}
			return logLineMsg{line: line, lines: lines, errs: errs}
		case err, ok := <-errs:
			if ok && err != nil {
				return logErrMsg{err: err}
			}
			return nil
		}
	}
}

func (m Model) handlePalette(cmd components.PaletteCommand) (tea.Model, tea.Cmd) {
	switch cmd.Name {
	case "dashboard":
		m.view = viewDashboard
		m.header = m.header.SetView("DASHBOARD")
		return m, tea.Batch(m.refreshContainersCmd())
	case "containers":
		m.view = viewContainers
		m.header = m.header.SetView("CONTAINERS")
		return m, nil
	case "images":
		m.view = viewImages
		m.header = m.header.SetView("IMAGES")
		return m, m.refreshImagesCmd()
	case "volumes":
		m.view = viewVolumes
		m.header = m.header.SetView("VOLUMES")
		return m, m.refreshVolumesCmd()
	case "networks":
		m.view = viewNetworks
		m.header = m.header.SetView("NETWORKS")
		return m, m.refreshNetworksCmd()
	case "security":
		m.view = viewSecurity
		m.header = m.header.SetView("SECURITY")
		return m, m.refreshSecurityCmd()
	case "diagnostics":
		m.view = viewDiagnostics
		m.header = m.header.SetView("DIAGNOSTICS")
		return m, m.refreshDiagnosticsCmd()
	case "events":
		m.view = viewDashboard
		m.header = m.header.SetView("DASHBOARD")
		return m, nil
	case "theme":
		if len(cmd.Args) > 0 {
			m.styles = themes.NewStyles(themes.ByName(cmd.Args[0]))
			m = m.restyle()
		}
		return m, nil
	case "refresh":
		return m, tea.Batch(m.refreshHostCmd(), m.refreshContainersCmd())
	case "help":
		m.helpOpen = true
		m.view = viewHelp
		m.header = m.header.SetView("HELP")
		return m, nil
	case "quit":
		m.cleanup()
		return m, tea.Quit
	case "compose":
		m.view = viewCompose
		m.header = m.header.SetView("COMPOSE")
		return m, m.refreshContainersCmd()
	case "contexts":
		m.statusMsg = "multi-host contexts arrive in a later milestone"
		m.list = m.list.SetStatus(m.statusMsg)
		return m, nil
	case "":
		return m, nil
	default:
		m.statusMsg = "unknown command: " + cmd.Raw
		m.list = m.list.SetStatus(m.statusMsg)
		return m, nil
	}
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case ":":
		m.palette = m.palette.Open()
		return m, nil
	case "tab":
		return m.cycleView(1)
	case "shift+tab":
		return m.cycleView(-1)
	case "ctrl+c":
		m.cleanup()
		return m, tea.Quit
	}

	switch m.view {
	case viewInspect:
		switch msg.String() {
		case "esc", "q":
			m.view = viewContainers
			m.header = m.header.SetView("CONTAINERS")
			return m, nil
		default:
			var cmd tea.Cmd
			m.inspect, cmd = m.inspect.Update(msg)
			return m, cmd
		}
	case viewDetail:
		switch msg.String() {
		case "esc", "q":
			back := m.detail.BackView()
			switch back {
			case "VOLUMES":
				m.view = viewVolumes
			case "NETWORKS":
				m.view = viewNetworks
			default:
				m.view = viewImages
			}
			m.header = m.header.SetView(back)
			return m, nil
		default:
			var cmd tea.Cmd
			m.detail, cmd = m.detail.Update(msg)
			return m, cmd
		}
	case viewLogs:
		switch msg.String() {
		case "esc", "q":
			if m.logCancel != nil {
				m.logCancel()
				m.logCancel = nil
			}
			m.view = viewContainers
			m.header = m.header.SetView("CONTAINERS")
			return m, nil
		default:
			var cmd tea.Cmd
			m.logs, cmd = m.logs.Update(msg)
			return m, cmd
		}
	case viewDashboard:
		switch msg.String() {
		case "q":
			m.cleanup()
			return m, tea.Quit
		case "?":
			m.helpOpen = true
			m.view = viewHelp
			m.header = m.header.SetView("HELP")
			return m, nil
		case "r":
			return m, tea.Batch(m.refreshHostCmd(), m.refreshContainersCmd())
		}
		return m, nil
	case viewImages, viewVolumes, viewNetworks, viewCompose, viewSecurity, viewDiagnostics:
		switch msg.String() {
		case "q":
			m.cleanup()
			return m, tea.Quit
		case "esc":
			m.view = viewContainers
			m.header = m.header.SetView("CONTAINERS")
			return m, nil
		case "r":
			switch m.view {
			case viewImages:
				return m, m.refreshImagesCmd()
			case viewVolumes:
				return m, m.refreshVolumesCmd()
			case viewNetworks:
				return m, m.refreshNetworksCmd()
			case viewCompose:
				return m, m.refreshContainersCmd()
			case viewSecurity:
				return m, m.refreshSecurityCmd()
			default:
				return m, m.refreshDiagnosticsCmd()
			}
		case "enter", "i":
			switch m.view {
			case viewImages:
				img, ok := m.images.SelectedImage()
				if !ok {
					return m, nil
				}
				m.view = viewDetail
				m.header = m.header.SetView("INSPECT")
				m.detail = m.detail.SetStyles(m.styles).Clear(detailscreen.KindImage, "IMAGES")
				m = m.layout()
				return m, m.inspectImageCmd(img.ID)
			case viewVolumes:
				vol, ok := m.volumes.SelectedVolume()
				if !ok {
					return m, nil
				}
				m.view = viewDetail
				m.header = m.header.SetView("INSPECT")
				m.detail = m.detail.SetStyles(m.styles).Clear(detailscreen.KindVolume, "VOLUMES")
				m = m.layout()
				return m, m.inspectVolumeCmd(vol.Name)
			case viewNetworks:
				net, ok := m.networks.SelectedNetwork()
				if !ok {
					return m, nil
				}
				m.view = viewDetail
				m.header = m.header.SetView("INSPECT")
				m.detail = m.detail.SetStyles(m.styles).Clear(detailscreen.KindNetwork, "NETWORKS")
				m = m.layout()
				return m, m.inspectNetworkCmd(net.ID)
			}
		}
		var cmd tea.Cmd
		switch m.view {
		case viewImages:
			m.images, cmd = m.images.Update(msg)
		case viewVolumes:
			m.volumes, cmd = m.volumes.Update(msg)
		case viewNetworks:
			m.networks, cmd = m.networks.Update(msg)
		case viewCompose:
			m.compose, cmd = m.compose.Update(msg)
		case viewSecurity:
			m.security, cmd = m.security.Update(msg)
		default:
			m.diags, cmd = m.diags.Update(msg)
		}
		return m, cmd
	}

	if m.list.Filtering() {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "q":
		m.cleanup()
		return m, tea.Quit
	case "?":
		m.helpOpen = true
		m.view = viewHelp
		m.header = m.header.SetView("HELP")
		return m, nil
	case "r":
		m.statusMsg = "refreshing…"
		m.list = m.list.SetStatus(m.statusMsg)
		return m, tea.Batch(m.refreshHostCmd(), m.refreshContainersCmd())
	case "enter", "i":
		c, ok := m.list.Selected()
		if !ok {
			return m, nil
		}
		m.view = viewInspect
		m.header = m.header.SetView("INSPECT")
		// Reuse the existing screen so width/height from the last
		// WindowSizeMsg/layout pass are preserved. New() would reset
		// them to 0 and shrink the panel until the next resize.
		m.inspect = m.inspect.SetStyles(m.styles).Clear()
		m = m.layout()
		cmds := []tea.Cmd{m.inspectCmd(c.ID)}
		if c.IsRunning() {
			cmds = append(cmds, m.refreshInspectStatsCmd(c.ID))
		}
		return m, tea.Batch(cmds...)
	case "l":
		c, ok := m.list.Selected()
		if !ok {
			return m, nil
		}
		return m, m.openLogsCmd(c.ID, c.Name)
	case "S":
		return m.openShell()
	case "s":
		return m.requestAction("start", false)
	case "x":
		return m.requestAction("stop", true)
	case "R":
		return m.requestAction("restart", true)
	case "d":
		// Prefer diagnostics shortcut per Agents.md; remove stays on destructive confirm via palette later.
		c, ok := m.list.Selected()
		if !ok {
			return m, nil
		}
		m.view = viewDiagnostics
		m.header = m.header.SetView("DIAGNOSTICS")
		id := c.ID
		return m, func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			items, err := m.deps.Diagnostics.AnalyzeContainer(ctx, id)
			return diagnosticsMsg{items: items, err: err}
		}
	case "D":
		return m.requestAction("remove", true)
	case "p":
		targets := m.list.Targets()
		if len(targets) == 0 {
			return m, nil
		}
		// Mixed sets: pause running, unpause paused; ignore others.
		paused, running := 0, 0
		for _, c := range targets {
			switch c.Status {
			case models.StatusPaused:
				paused++
			case models.StatusRunning:
				running++
			}
		}
		if paused > 0 && running == 0 {
			return m.requestAction("unpause", false)
		}
		return m.requestAction("pause", false)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) openShell() (tea.Model, tea.Cmd) {
	c, ok := m.list.Selected()
	if !ok {
		return m, nil
	}
	if !c.IsRunning() {
		m.statusMsg = "shell requires a running container"
		m.list = m.list.SetStatus(m.statusMsg)
		return m, nil
	}
	m.statusMsg = "opening shell in " + c.Name
	m.list = m.list.SetStatus(m.statusMsg)
	cmd := m.deps.Shell.NewCommand(c.ID, []string{"/bin/sh"})
	return m, tea.Exec(cmd, func(err error) tea.Msg {
		return shellDoneMsg{err: err}
	})
}

func (m Model) cycleView(dir int) (tea.Model, tea.Cmd) {
	// Must match header tab order exactly.
	order := []viewState{
		viewDashboard,
		viewContainers,
		viewCompose,
		viewImages,
		viewNetworks,
		viewVolumes,
		viewSecurity,
		viewDiagnostics,
	}
	idx := 0
	found := false
	for i, v := range order {
		if v == m.view {
			idx = i
			found = true
			break
		}
	}
	if !found {
		// Inspect/logs/help are not tabs — Tab returns to containers first.
		if dir > 0 {
			idx = 1 // containers
		} else {
			idx = 0 // dashboard
		}
		m.view = order[idx]
	} else {
		idx = (idx + dir + len(order)) % len(order)
		m.view = order[idx]
	}
	switch m.view {
	case viewDashboard:
		m.header = m.header.SetView("DASHBOARD")
		return m, m.refreshContainersCmd()
	case viewContainers:
		m.header = m.header.SetView("CONTAINERS")
		return m, nil
	case viewCompose:
		m.header = m.header.SetView("COMPOSE")
		return m, m.refreshContainersCmd()
	case viewImages:
		m.header = m.header.SetView("IMAGES")
		return m, m.refreshImagesCmd()
	case viewNetworks:
		m.header = m.header.SetView("NETWORKS")
		return m, m.refreshNetworksCmd()
	case viewVolumes:
		m.header = m.header.SetView("VOLUMES")
		return m, m.refreshVolumesCmd()
	case viewSecurity:
		m.header = m.header.SetView("SECURITY")
		return m, m.refreshSecurityCmd()
	case viewDiagnostics:
		m.header = m.header.SetView("DIAGNOSTICS")
		return m, m.refreshDiagnosticsCmd()
	}
	return m, nil
}

func (m Model) requestAction(action string, confirm bool) (tea.Model, tea.Cmd) {
	targets := m.list.Targets()
	if len(targets) == 0 {
		return m, nil
	}
	if confirm && m.deps.Config.Confirm.DestructiveActions {
		m.pendingAction = action
		m.pendingBulk = targets
		m.pendingID, m.pendingName = targets[0].ID, targets[0].Name
		var msg string
		if len(targets) == 1 {
			c := targets[0]
			msg = fmt.Sprintf("About to %s container %q (%s).", action, c.Name, c.ShortID())
		} else {
			names := make([]string, 0, len(targets))
			for _, c := range targets {
				names = append(names, c.Name)
			}
			msg = fmt.Sprintf("About to %s %d containers:\n%s", action, len(targets), strings.Join(names, ", "))
		}
		if action == "remove" {
			msg += " Persistent volumes will NOT be deleted."
		}
		m.confirm = m.confirm.Show(action+":bulk", "Confirm "+strings.ToUpper(action), msg)
		return m, nil
	}
	return m, m.runBulkActionCmd(action, targets)
}

func (m Model) layout() Model {
	m.header = m.header.SetSize(m.width).SetHost(m.host).SetError(m.errText)
	bodyHeight := m.height - 8
	if bodyHeight < 8 {
		bodyHeight = 8
	}
	m.list = m.list.SetSize(m.width, bodyHeight).SetStyles(m.styles)
	m.inspect = m.inspect.SetSize(m.width, bodyHeight).SetStyles(m.styles)
	m.detail = m.detail.SetSize(m.width, bodyHeight).SetStyles(m.styles)
	m.logs = m.logs.SetSize(m.width, bodyHeight).SetStyles(m.styles)
	m.dash = m.dash.SetSize(m.width, bodyHeight).SetStyles(m.styles)
	m.images = m.images.SetSize(m.width, bodyHeight).SetStyles(m.styles)
	m.volumes = m.volumes.SetSize(m.width, bodyHeight).SetStyles(m.styles)
	m.networks = m.networks.SetSize(m.width, bodyHeight).SetStyles(m.styles)
	m.compose = m.compose.SetSize(m.width, bodyHeight).SetStyles(m.styles)
	m.security = m.security.SetSize(m.width, bodyHeight).SetStyles(m.styles)
	m.diags = m.diags.SetSize(m.width, bodyHeight).SetStyles(m.styles)
	m.confirm = m.confirm.SetSize(m.width).SetStyles(m.styles)
	m.palette = m.palette.SetSize(m.width).SetStyles(m.styles)
	return m
}

func (m Model) restyle() Model {
	m.header = m.header.SetStyles(m.styles)
	return m.layout()
}

func (m *Model) cleanup() {
	if m.logCancel != nil {
		m.logCancel()
		m.logCancel = nil
	}
	if m.evtCancel != nil {
		m.evtCancel()
		m.evtCancel = nil
	}
}

func (m Model) View() tea.View {
	if m.width == 0 || m.height == 0 {
		v := tea.NewView("Loading DOKI…")
		v.AltScreen = true
		return v
	}
	if m.width < 60 || m.height < 18 {
		msg := m.styles.StatusWarn.Render("Terminal too small for DOKI.\nResize to at least 60x18.")
		v := tea.NewView(m.styles.App.Width(m.width).Height(m.height).Render(msg))
		v.AltScreen = true
		return v
	}

	var body string
	switch {
	case m.helpOpen || m.view == viewHelp:
		body = m.renderHelp()
	case m.view == viewInspect:
		body = m.inspect.View()
	case m.view == viewDetail:
		body = m.detail.View()
	case m.view == viewLogs:
		body = m.logs.View()
	case m.view == viewDashboard:
		body = m.dash.View()
	case m.view == viewImages:
		body = m.images.View()
	case m.view == viewVolumes:
		body = m.volumes.View()
	case m.view == viewNetworks:
		body = m.networks.View()
	case m.view == viewCompose:
		body = m.compose.View()
	case m.view == viewSecurity:
		body = m.security.View()
	case m.view == viewDiagnostics:
		body = m.diags.View()
	default:
		body = m.list.View()
	}

	footerBindings := append(m.keys.Containers, m.keys.Global...)
	switch m.view {
	case viewInspect, viewDetail:
		footerBindings = append(m.keys.Inspect, m.keys.Global...)
	case viewLogs:
		footerBindings = []keymap.Binding{{Keys: "esc", Help: "Back"}, {Keys: "f", Help: "Follow"}, {Keys: "/", Help: "Search"}}
	case viewImages, viewVolumes, viewNetworks:
		footerBindings = []keymap.Binding{
			{Keys: "j/k", Help: "Navigate"},
			{Keys: "enter", Help: "Inspect"},
			{Keys: "r", Help: "Refresh"},
			{Keys: "esc", Help: "Back"},
			{Keys: "q", Help: "Quit"},
		}
	}
	if m.confirm.Active {
		footerBindings = m.keys.Confirm
	}
	footer := m.styles.Footer.Width(m.width).Render(keymap.FormatFooter(footerBindings) + "  [:] Palette")

	parts := []string{m.header.SetView(m.headerViewLabel()).View(), body, footer}
	if m.palette.Active() {
		parts = append(parts, m.palette.View())
	}
	if m.confirm.Active {
		parts = append(parts, m.confirm.View())
	}
	content := strings.Join(parts, "\n")
	framed := m.styles.App.Width(m.width).Height(m.height).Render(content)
	v := tea.NewView(framed)
	v.AltScreen = true
	return v
}

func (m Model) headerViewLabel() string {
	switch {
	case m.helpOpen:
		return "HELP"
	case m.view == viewInspect, m.view == viewDetail:
		return "INSPECT"
	case m.view == viewLogs:
		return "LOGS"
	case m.view == viewDashboard:
		return "DASHBOARD"
	case m.view == viewImages:
		return "IMAGES"
	case m.view == viewVolumes:
		return "VOLUMES"
	case m.view == viewNetworks:
		return "NETWORKS"
	case m.view == viewCompose:
		return "COMPOSE"
	case m.view == viewSecurity:
		return "SECURITY"
	case m.view == viewDiagnostics:
		return "DIAGNOSTICS"
	default:
		return "CONTAINERS"
	}
}

func (m Model) renderHelp() string {
	var b strings.Builder
	b.WriteString(m.styles.Title.Render("KEYBOARD SHORTCUTS") + "\n\n")
	b.WriteString(m.styles.Brand.Render("Navigation") + "\n")
	b.WriteString("  Tab / Shift+Tab  Cycle views\n  :                 Command palette\n  j/k               Move\n\n")
	b.WriteString(m.styles.Brand.Render("Containers") + "\n")
	for _, item := range m.keys.Containers {
		_, _ = fmt.Fprintf(&b, "  %-10s %s\n", item.Keys, item.Help)
	}
	b.WriteString("\n  space marks for bulk start/stop/restart/remove; A marks all, U/esc clears\n\n")
	b.WriteString(m.styles.Help.Render("[esc] Close help"))
	return m.styles.Panel.Width(m.width).Height(m.height - 8).Render(b.String())
}

func friendlyDockerError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "permission denied"):
		return "Docker permission denied — is your user in the docker group?"
	case strings.Contains(lower, "cannot connect"), strings.Contains(lower, "no such file"), strings.Contains(lower, "connection refused"):
		return "Cannot connect to Docker Engine — is the daemon running?"
	default:
		if len(msg) > 120 {
			return msg[:117] + "..."
		}
		return msg
	}
}

// Run starts the Bubble Tea program.
func Run(deps Dependencies) error {
	m := New(deps)
	p := tea.NewProgram(m)
	_, err := p.Run()
	m.cleanup()
	return err
}
