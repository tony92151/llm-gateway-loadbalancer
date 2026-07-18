package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/tony92151/llm-gateway-loadbalancer/internal/accounting"
	"github.com/tony92151/llm-gateway-loadbalancer/internal/config"
	"github.com/tony92151/llm-gateway-loadbalancer/internal/health"
	"github.com/tony92151/llm-gateway-loadbalancer/internal/httpserver"
	"github.com/tony92151/llm-gateway-loadbalancer/internal/proxy"
	"github.com/tony92151/llm-gateway-loadbalancer/internal/selector"
	"github.com/tony92151/llm-gateway-loadbalancer/internal/store"
)

type App struct {
	cfg          config.Config
	db           *store.DB
	proxyHandler *proxy.Handler
	adminHandler http.Handler
	health       *health.Checker
}

func New(cfg config.Config) (*App, error) {
	if err := os.MkdirAll(filepath.Dir(cfg.Database.Path), 0o755); err != nil && cfg.Database.Path != ":memory:" {
		return nil, err
	}
	db, err := store.Open(cfg.Database.Path, cfg.Database.WALMode, cfg.Database.MaxOpenConn, cfg.Database.MaxIdleConn)
	if err != nil {
		return nil, err
	}
	if err := db.Migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}

	proxyHandler, err := proxy.NewHandlerWithError(BuildProxyConfig(cfg, proxy.StoreRecorder{DB: db}))
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := db.UpsertKeyStates(proxyHandler.Keys()); err != nil {
		_ = db.Close()
		return nil, err
	}
	healthChecker, err := health.NewChecker(health.Config{
		BaseURL:  cfg.Upstream.BaseURL,
		Pool:     proxyHandler.Pool(),
		Cooldown: cfg.Selector.CooldownBase,
	})
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	provider := adminProvider{proxy: proxyHandler, db: db}
	return &App{
		cfg:          cfg,
		db:           db,
		proxyHandler: proxyHandler,
		adminHandler: httpserver.NewAdminHandler(provider, cfg.Admin.MonitorUI),
		health:       healthChecker,
	}, nil
}

func BuildProxyConfig(cfg config.Config, recorder proxy.Recorder) proxy.Config {
	keys := make([]selector.Key, 0, len(cfg.Upstream.Keys))
	for _, key := range cfg.Upstream.Keys {
		keys = append(keys, selector.Key{
			Label:    key.Label,
			Secret:   key.Key,
			Weight:   key.Weight,
			RPMLimit: key.RPMLimit,
			TPMLimit: key.TPMLimit,
			Enabled:  key.Enabled,
		})
	}

	prices := make(map[string]accounting.Pricing, len(cfg.Upstream.Models))
	for _, model := range cfg.Upstream.Models {
		if !model.Enabled {
			continue
		}
		prices[model.Name] = accounting.Pricing{
			InputPer1M:       model.Pricing.InputPer1M,
			OutputPer1M:      model.Pricing.OutputPer1M,
			CachedInputPer1M: model.Pricing.CachedInputPer1M,
		}
	}

	return proxy.Config{
		BaseURL:         cfg.Upstream.BaseURL,
		Keys:            keys,
		Strategy:        cfg.Selector.Strategy,
		MaxRetries:      cfg.Selector.MaxRetries,
		CooldownBase:    cfg.Selector.CooldownBase,
		Timeout:         cfg.Upstream.Timeout,
		MaxIdleConns:    cfg.Upstream.MaxIdleConns,
		MaxConnsPerHost: cfg.Upstream.MaxConnsPerHost,
		Prices:          prices,
		Recorder:        recorder,
	}
}

func (a *App) Run(ctx context.Context) error {
	defer a.db.Close()
	stopSchedulers := a.startSchedulers(ctx)
	defer stopSchedulers()

	proxyServer := &http.Server{
		Addr:         net.JoinHostPort(a.cfg.Server.Host, fmt.Sprint(a.cfg.Server.Port)),
		Handler:      a.proxyHandler,
		ReadTimeout:  a.cfg.Server.ReadTimeout,
		WriteTimeout: a.cfg.Server.WriteTimeout,
		IdleTimeout:  a.cfg.Server.IdleTimeout,
	}
	adminPort := a.cfg.Admin.Port
	adminServer := &http.Server{
		Addr:         net.JoinHostPort(a.cfg.Admin.Host, fmt.Sprint(adminPort)),
		Handler:      a.adminHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	errCh := make(chan error, 2)
	go func() {
		errCh <- serve(proxyServer)
	}()
	go func() {
		errCh <- serve(adminServer)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := errors.Join(proxyServer.Shutdown(shutdownCtx), adminServer.Shutdown(shutdownCtx))
		return errors.Join(err, ctx.Err())
	case err := <-errCh:
		return err
	}
}

func (a *App) startSchedulers(ctx context.Context) func() {
	healthInterval := a.cfg.Selector.HealthCheckInterval
	if healthInterval <= 0 {
		healthInterval = 30 * time.Second
	}
	healthCtx, cancelHealth := context.WithCancel(ctx)
	go a.runHealthChecks(healthCtx, healthInterval)

	c := cron.New()
	_, _ = c.AddFunc("0 * * * *", func() {
		hour := time.Now().UTC().Add(-time.Hour).Truncate(time.Hour)
		_ = a.db.AggregateHourly(hour)
	})
	c.Start()

	return func() {
		cancelHealth()
		stopCtx := c.Stop()
		<-stopCtx.Done()
	}
}

func (a *App) runHealthChecks(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = a.health.CheckOnce(ctx)
			_ = a.db.UpsertKeyStates(a.proxyHandler.Keys())
		}
	}
}

func serve(server *http.Server) error {
	err := server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

type adminProvider struct {
	proxy *proxy.Handler
	db    *store.DB
}

func (p adminProvider) Keys() []selector.Key {
	return p.proxy.Keys()
}

func (p adminProvider) RecentRequests(limit int) ([]store.RequestLog, error) {
	return p.db.RecentRequestLogs(limit)
}

func (p adminProvider) SummarySince(since time.Time) (store.Summary, error) {
	return p.db.SummarySince(since)
}

func (p adminProvider) Dashboard(window time.Duration) (httpserver.DashboardResponse, error) {
	stats, err := p.db.DashboardSince(time.Now().UTC().Add(-window), 20)
	if err != nil {
		return httpserver.DashboardResponse{}, err
	}

	usageByKey := make(map[string]store.DashboardKeyStats, len(stats.Keys))
	for _, key := range stats.Keys {
		usageByKey[key.Label] = key
	}

	keys := p.proxy.Keys()
	responseKeys := make([]httpserver.DashboardKey, 0, len(keys))
	for _, key := range keys {
		usage := usageByKey[key.Label]
		responseKeys = append(responseKeys, httpserver.DashboardKey{
			Label:         key.Label,
			Enabled:       key.Enabled,
			InFlight:      key.InFlight,
			CooldownUntil: key.CooldownUntil,
			LastError:     key.LastError,
			Requests:      usage.Requests,
			Errors:        usage.Errors,
			Tokens:        usage.Tokens,
			CostUSD:       usage.CostUSD,
		})
		delete(usageByKey, key.Label)
	}
	for label, usage := range usageByKey {
		responseKeys = append(responseKeys, httpserver.DashboardKey{
			Label:    label,
			Requests: usage.Requests,
			Errors:   usage.Errors,
			Tokens:   usage.Tokens,
			CostUSD:  usage.CostUSD,
		})
	}

	return httpserver.DashboardResponse{
		Overview: httpserver.DashboardOverview{
			Requests:     stats.Overview.Requests,
			SuccessRate:  stats.Overview.SuccessRate,
			Errors:       stats.Overview.Errors,
			InputTokens:  stats.Overview.InputTokens,
			OutputTokens: stats.Overview.OutputTokens,
			CostUSD:      stats.Overview.CostUSD,
			AvgLatencyMS: stats.Overview.AvgLatencyMS,
		},
		Keys:         responseKeys,
		RecentErrors: stats.RecentErrors,
	}, nil
}
