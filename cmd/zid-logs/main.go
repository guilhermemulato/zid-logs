package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"zid-logs/internal/config"
	"zid-logs/internal/registry"
	"zid-logs/internal/rotate"
	"zid-logs/internal/shipper"
	"zid-logs/internal/state"
	"zid-logs/internal/status"
)

const version = "0.1.7"

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

	rotateInterval := time.Duration(cfg.IntervalRotateSeconds) * time.Second
	shipInterval := time.Duration(cfg.IntervalShipSeconds) * time.Second

	if rotateInterval <= 0 {
		rotateInterval = 300 * time.Second
	}
	if shipInterval <= 0 {
		shipInterval = 60 * time.Second
	}

	var mu sync.Mutex
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reload := make(chan os.Signal, 1)
	stop := make(chan os.Signal, 1)
	trapSignals(reload, stop)

	rotateTicker := time.NewTicker(rotateInterval)
	shipTicker := time.NewTicker(shipInterval)
	defer rotateTicker.Stop()
	defer shipTicker.Stop()

	log.Printf("zid-logs iniciado")

	for {
		select {
		case <-rotateTicker.C:
			mu.Lock()
			inputs = refreshInputs(inputs)
			if err := rotateAll(cfg, inputs); err != nil {
				log.Printf("erro na rotacao: %v", err)
			}
			mu.Unlock()
		case <-shipTicker.C:
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
			mu.Unlock()
		case <-stop:
			log.Printf("zid-logs encerrando")
			return
		case <-ctx.Done():
			return
		}
	}
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

	if err := rotateAll(cfg, inputs); err != nil {
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
	cfg, inputs, st, err := loadAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "erro ao carregar configuracoes: %v\n", err)
		os.Exit(1)
	}
	defer st.Close()
	_ = cfg

	payload := status.Build(inputs, st, "")
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

func rotateAll(cfg config.Config, inputs []registry.LogInput) error {
	for _, input := range inputs {
		policy := rotate.ResolvePolicy(cfg.Defaults, input.Policy)
		rotated, err := rotate.RotateIfNeeded(input.Path, policy)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		if rotated {
			log.Printf("rotacionado %s", input.Path)
		}
	}
	return nil
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
