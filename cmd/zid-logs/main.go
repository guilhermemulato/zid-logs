package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"zid-logs/internal/config"
	"zid-logs/internal/registry"
	"zid-logs/internal/rotate"
	"zid-logs/internal/shipper"
	"zid-logs/internal/state"
	"zid-logs/internal/status"

	bolt "go.etcd.io/bbolt"
)

const version = "0.1.10.16"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "run":
		runCmd()
	case "rotate":
		rotateCmd()
	case "ship":
		shipCmd()
	case "status":
		statusCmd()
	case "validate":
		validateCmd()
	case "version", "-version", "--version", "-v":
		fmt.Printf("zid-logs version %s\n", version)
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Println("Usage: zid-logs <run|rotate|ship|status|validate|version>")
}

func runCmd() {
	logger := newLogger()
	defer logger.Close()
	log.SetOutput(logger)

	cfg, inputs, st, err := loadAll()
	if err != nil {
		log.Printf("erro ao carregar configuracoes: %v", err)
		os.Exit(1)
	}
	defer st.Close()

	if !cfg.Enabled {
		log.Printf("zid-logs desabilitado")
		return
	}

	var mu sync.Mutex
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reload := make(chan os.Signal, 1)
	stop := make(chan os.Signal, 1)
	trapSignals(reload, stop)

	rotateSched := startRotateScheduler(cfg)
	shipTicker := startShipTicker(cfg)
	defer rotateSched.Stop()
	defer shipTicker.Stop()

	log.Printf("zid-logs iniciado")
	rotateIfDue(cfg, inputs, st)

	for {
		select {
		case <-rotateSched.C:
			mu.Lock()
			inputs = refreshInputs(inputs)
			if cfg.RotateAt != "" {
				rotateIfDue(cfg, inputs, st)
			} else if err := rotateAll(cfg, inputs, st, false); err != nil {
				log.Printf("erro na rotacao: %v", err)
			}
			mu.Unlock()
		case <-shipTicker.C():
			mu.Lock()
			inputs = refreshInputs(inputs)
			if err := shipAll(ctx, cfg, inputs, st); err != nil {
				log.Printf("erro no envio: %v", err)
			}
			mu.Unlock()
		case <-reload:
			mu.Lock()
			_ = st.Close()
			cfg, inputs, st, err = loadAll()
			if err != nil {
				log.Printf("erro ao recarregar configuracoes: %v", err)
			}
			rotateSched.Update(cfg)
			shipTicker.Update(cfg)
			mu.Unlock()
		case <-stop:
			log.Printf("zid-logs encerrando")
			return
		case <-ctx.Done():
			return
		}
	}
}

type rotateScheduler struct {
	C    <-chan time.Time
	stop chan struct{}
	done chan struct{}
}

func startRotateScheduler(cfg config.Config) *rotateScheduler {
	stop := make(chan struct{})
	done := make(chan struct{})
	out := make(chan time.Time)

	go func() {
		defer close(done)
		var ticker *time.Ticker

		if cfg.RotateAt != "" {
			ticker = time.NewTicker(time.Minute)
		} else {
			interval := time.Duration(cfg.IntervalRotateSeconds) * time.Second
			if interval <= 0 {
				interval = 300 * time.Second
			}
			ticker = time.NewTicker(interval)
		}

		for {
			select {
			case <-stop:
				if ticker != nil {
					ticker.Stop()
				}
				return
			default:
			}

			select {
			case t := <-ticker.C:
				out <- t
			case <-stop:
				ticker.Stop()
				return
			}
		}
	}()

	return &rotateScheduler{C: out, stop: stop, done: done}
}

func (s *rotateScheduler) Stop() {
	close(s.stop)
	<-s.done
}

func (s *rotateScheduler) Update(cfg config.Config) {
	s.Stop()
	newSched := startRotateScheduler(cfg)
	*s = *newSched
}

type shipIntervalTicker struct {
	ticker *time.Ticker
	ch     chan time.Time
	stop   chan struct{}
	done   chan struct{}
}

func startShipTicker(cfg config.Config) *shipIntervalTicker {
	ch := make(chan time.Time)
	stop := make(chan struct{})
	done := make(chan struct{})
	ticker := time.NewTicker(resolveShipInterval(cfg))
	go func() {
		defer close(done)
		for {
			select {
			case t := <-ticker.C:
				ch <- t
			case <-stop:
				ticker.Stop()
				return
			}
		}
	}()
	return &shipIntervalTicker{ticker: ticker, ch: ch, stop: stop, done: done}
}

func (t *shipIntervalTicker) C() <-chan time.Time {
	return t.ch
}

func (t *shipIntervalTicker) Stop() {
	close(t.stop)
	<-t.done
	close(t.ch)
}

func (t *shipIntervalTicker) Update(cfg config.Config) {
	t.Stop()
	newTicker := startShipTicker(cfg)
	*t = *newTicker
}

func resolveShipInterval(cfg config.Config) time.Duration {
	if cfg.ShipIntervalHours > 0 {
		return time.Duration(cfg.ShipIntervalHours) * time.Hour
	}
	if cfg.IntervalShipSeconds > 0 {
		return time.Duration(cfg.IntervalShipSeconds) * time.Second
	}
	return time.Hour
}

func nextRotateTime(now time.Time, rotateAt string) (time.Time, error) {
	hour, minute, err := parseRotateAt(rotateAt)
	if err != nil {
		return time.Time{}, err
	}
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next, nil
}

func parseRotateAt(value string) (int, int, error) {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) == 0 || len(parts) > 2 {
		return 0, 0, fmt.Errorf("rotate_at invalido")
	}
	hour, err := strconv.Atoi(parts[0])
	if err != nil || hour < 0 || hour > 23 {
		return 0, 0, fmt.Errorf("hora invalida")
	}
	minute := 0
	if len(parts) == 2 {
		minute, err = strconv.Atoi(parts[1])
		if err != nil || minute < 0 || minute > 59 {
			return 0, 0, fmt.Errorf("minuto invalido")
		}
	}
	return hour, minute, nil
}

func rotateCmd() {
	logger := newLogger()
	defer logger.Close()
	log.SetOutput(logger)

	cfg, inputs, st, err := loadAll()
	if err != nil {
		log.Printf("erro ao carregar configuracoes: %v", err)
		os.Exit(1)
	}
	defer st.Close()

	if err := rotateAll(cfg, inputs, st, true); err != nil {
		log.Printf("erro na rotacao: %v", err)
		os.Exit(1)
	}
}

func shipCmd() {
	logger := newLogger()
	defer logger.Close()
	log.SetOutput(logger)

	cfg, inputs, st, err := loadAll()
	if err != nil {
		log.Printf("erro ao carregar configuracoes: %v", err)
		os.Exit(1)
	}
	defer st.Close()

	ctx := context.Background()
	if err := shipAll(ctx, cfg, inputs, st); err != nil {
		log.Printf("erro no envio: %v", err)
		os.Exit(1)
	}
}

func statusCmd() {
	cfg, inputs, st, err := loadAllReadOnly()
	if err != nil {
		fmt.Fprintf(os.Stderr, "erro ao carregar configuracoes: %v\n", err)
		os.Exit(1)
	}
	if st != nil {
		defer st.Close()
	}

	payload := status.Build(cfg, inputs, st, "")
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "erro ao gerar status: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}

func validateCmd() {
	cfg, inputs, st, err := loadAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "erro ao carregar configuracoes: %v\n", err)
		os.Exit(1)
	}
	defer st.Close()

	var problems []string
	if cfg.Enabled && cfg.Endpoint == "" {
		problems = append(problems, "endpoint nao configurado")
	}
	if cfg.Enabled && cfg.AuthToken == "" {
		problems = append(problems, "auth_token nao configurado")
	}

	for _, input := range inputs {
		if input.Package == "" || input.LogID == "" || input.Path == "" {
			problems = append(problems, fmt.Sprintf("input invalido em %s", input.Source))
		}
	}

	if len(problems) > 0 {
		for _, p := range problems {
			fmt.Fprintf(os.Stderr, "- %s\n", p)
		}
		os.Exit(1)
	}
	fmt.Println("ok")
}

func loadAll() (config.Config, []registry.LogInput, *state.State, error) {
	cfg, err := config.LoadConfig(config.DefaultConfigPath)
	if err != nil {
		return config.Config{}, nil, nil, err
	}
	cfg, err = config.EnsureDeviceID(cfg)
	if err != nil {
		return config.Config{}, nil, nil, err
	}

	inputs, err := loadInputsSafe(config.DefaultInputsDir)
	if err != nil {
		return config.Config{}, nil, nil, err
	}

	if err := os.MkdirAll(filepath.Dir(config.StateDBPath), 0755); err != nil {
		return config.Config{}, nil, nil, err
	}
	st, err := state.Open(config.StateDBPath)
	if err != nil {
		return config.Config{}, nil, nil, err
	}

	return cfg, inputs, st, nil
}

func loadAllReadOnly() (config.Config, []registry.LogInput, *state.State, error) {
	cfg, err := config.LoadConfig(config.DefaultConfigPath)
	if err != nil {
		return config.Config{}, nil, nil, err
	}
	cfg, err = config.EnsureDeviceID(cfg)
	if err != nil {
		return config.Config{}, nil, nil, err
	}

	inputs, err := loadInputsSafe(config.DefaultInputsDir)
	if err != nil {
		return config.Config{}, nil, nil, err
	}

	if err := os.MkdirAll(filepath.Dir(config.StateDBPath), 0755); err != nil {
		return config.Config{}, nil, nil, err
	}
	st, err := state.OpenReadOnly(config.StateDBPath)
	if err != nil {
		if errors.Is(err, bolt.ErrTimeout) || strings.Contains(err.Error(), "timeout") {
			return cfg, inputs, nil, nil
		}
		return config.Config{}, nil, nil, err
	}

	return cfg, inputs, st, nil
}

func loadInputsSafe(dir string) ([]registry.LogInput, error) {
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return []registry.LogInput{}, nil
		}
		return nil, err
	}
	return registry.LoadInputs(dir)
}

func refreshInputs(current []registry.LogInput) []registry.LogInput {
	inputs, err := loadInputsSafe(config.DefaultInputsDir)
	if err != nil {
		return current
	}
	return inputs
}

func rotateAll(cfg config.Config, inputs []registry.LogInput, st *state.State, force bool) error {
	for _, input := range inputs {
		rotated, err := rotateOne(cfg, input, st, force)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		if rotated {
			if err := notifyPostRotate(input); err != nil {
				log.Printf("post-rotate falhou %s: %v", input.Path, err)
			}
			log.Printf("rotacionado %s", input.Path)
		}
	}
	return nil
}

func rotateOne(cfg config.Config, input registry.LogInput, st *state.State, force bool) (bool, error) {
	policy := rotate.ResolvePolicy(cfg.Defaults, input.Policy)
	var rotated bool
	var err error
	if force {
		rotated, err = rotate.ForceRotate(input.Path, policy)
	} else {
		rotated, err = rotate.RotateIfNeeded(input.Path, policy)
	}
	if err != nil {
		return false, err
	}
	if rotated && st != nil {
		cp, ok, err := st.GetCheckpoint(input.Package, input.LogID, input.Path)
		if err == nil {
			if !ok {
				cp = state.Checkpoint{
					Package: input.Package,
					LogID:   input.LogID,
					Path:    input.Path,
				}
			}
			cp.LastRotateAt = time.Now().Unix()
			_ = st.SaveCheckpoint(cp)
		}
	}
	if rotated {
		if err := notifyPostRotate(input); err != nil {
			log.Printf("post-rotate falhou %s: %v", input.Path, err)
		}
	}
	return rotated, nil
}

func rotateIfDue(cfg config.Config, inputs []registry.LogInput, st *state.State) {
	if cfg.RotateAt == "" {
		return
	}
	hour, minute, err := parseRotateAt(cfg.RotateAt)
	if err != nil {
		log.Printf("rotate_at invalido: %v", err)
		return
	}
	now := time.Now()
	scheduled := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if now.Before(scheduled) {
		return
	}
	for _, input := range inputs {
		var lastRotate int64
		if st != nil {
			cp, ok, err := st.GetCheckpoint(input.Package, input.LogID, input.Path)
			if err == nil && ok {
				lastRotate = cp.LastRotateAt
			}
		}
		if lastRotate >= scheduled.Unix() {
			continue
		}

		rotated, err := rotateScheduled(cfg, input, st, scheduled)
		if err != nil {
			log.Printf("erro na rotacao: %v", err)
			continue
		}
		if rotated {
			log.Printf("rotacao agendada %s", input.Path)
		}
	}
}

func notifyPostRotate(input registry.LogInput) error {
	if input.PostRotateCommand != "" {
		cmd := exec.Command("/bin/sh", "-c", input.PostRotateCommand)
		return cmd.Run()
	}
	if input.PostRotatePidfile != "" {
		pidData, err := os.ReadFile(input.PostRotatePidfile)
		if err != nil {
			return err
		}
		pidStr := strings.TrimSpace(string(pidData))
		if pidStr == "" {
			return fmt.Errorf("pidfile vazio")
		}
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			return err
		}
		signal := resolveSignal(input.PostRotateSignal)
		return syscall.Kill(pid, signal)
	}
	if input.PostRotateMatch != "" {
		signal := resolveSignal(input.PostRotateSignal)
		return signalByMatch(input.PostRotateMatch, signal)
	}
	return nil
}

func resolveSignal(name string) syscall.Signal {
	switch strings.ToUpper(strings.TrimSpace(name)) {
	case "HUP":
		return syscall.SIGHUP
	case "TERM":
		return syscall.SIGTERM
	case "USR1":
		return syscall.SIGUSR1
	case "USR2":
		return syscall.SIGUSR2
	}
	return syscall.SIGHUP
}

func signalByMatch(match string, sig syscall.Signal) error {
	out, err := exec.Command("/usr/bin/pgrep", "-f", match).Output()
	if err != nil {
		return err
	}
	pids := strings.Fields(string(out))
	if len(pids) == 0 {
		return fmt.Errorf("nenhum pid encontrado")
	}
	for _, pidStr := range pids {
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}
		_ = syscall.Kill(pid, sig)
	}
	return nil
}

func rotateScheduled(cfg config.Config, input registry.LogInput, st *state.State, scheduled time.Time) (bool, error) {
	policy := rotate.ResolvePolicy(cfg.Defaults, input.Policy)
	var rotated bool
	var err error
	if input.TimestampLayout != "" {
		rotated, err = rotate.RotateByTimestampCut(input.Path, policy, input.TimestampLayout, scheduled)
	} else {
		rotated, err = rotate.ForceRotate(input.Path, policy)
	}
	if err != nil {
		return false, err
	}
	if rotated && st != nil {
		cp, ok, err := st.GetCheckpoint(input.Package, input.LogID, input.Path)
		if err == nil {
			if !ok {
				cp = state.Checkpoint{
					Package: input.Package,
					LogID:   input.LogID,
					Path:    input.Path,
				}
			}
			cp.LastRotateAt = time.Now().Unix()
			_ = st.SaveCheckpoint(cp)
		}
	}
	return rotated, nil
}

func shipAll(ctx context.Context, cfg config.Config, inputs []registry.LogInput, st *state.State) error {
	if os.Getenv("ZID_LOGS_DRY_RUN") == "1" {
		log.Printf("ZID_LOGS_DRY_RUN=1, envio ignorado")
		return nil
	}

	for _, input := range inputs {
		if input.Policy.ShipEnabled != nil && !*input.Policy.ShipEnabled {
			continue
		}
		_, err := shipper.ShipOnce(ctx, input, cfg, st)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
	}
	return nil
}

type logFile struct {
	file *os.File
}

func newLogger() *logFile {
	path := "/var/log/zid-logs.log"
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return &logFile{file: os.Stdout}
	}
	return &logFile{file: file}
}

func (l *logFile) Write(p []byte) (int, error) {
	if l == nil || l.file == nil {
		return len(p), nil
	}
	return l.file.Write(p)
}

func (l *logFile) Close() {
	if l == nil || l.file == nil || l.file == os.Stdout {
		return
	}
	_ = l.file.Close()
}

func trapSignals(reload chan os.Signal, stop chan os.Signal) {
	signal.Notify(reload, syscall.SIGHUP)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
}
