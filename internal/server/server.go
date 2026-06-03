package server

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"time"

	"github.com/d--j/go-milter/mailfilter"
)

type Config struct {
	Network         string
	Address         string
	SocketMode      os.FileMode
	SocketUser      string
	SocketGroup     string
	ShutdownTimeout time.Duration
	Now             func() time.Time
	Logger          *log.Logger
}

func Run(ctx context.Context, cfg Config) error {
	cfg = cfg.withDefaults()
	if cfg.Network == "unix" {
		if err := prepareUnixSocket(cfg.Address); err != nil {
			return err
		}
	}

	filter, err := mailfilter.New(
		cfg.Network,
		cfg.Address,
		utcDateDecision(cfg.Now),
		mailfilter.WithDecisionAt(mailfilter.DecisionAtEndOfHeaders),
		mailfilter.WithErrorHandling(mailfilter.AcceptWhenError),
		mailfilter.WithHeader(512, mailfilter.TruncateWhenTooBig),
		mailfilter.WithoutBody(),
	)
	if err != nil {
		return err
	}

	if cfg.Network == "unix" {
		if err := setSocketAccess(cfg.Address, cfg.SocketMode, cfg.SocketUser, cfg.SocketGroup); err != nil {
			filter.Close()
			return err
		}
	}

	cfg.Logger.Printf("utc-milter listening on %s:%s", filter.Addr().Network(), filter.Addr().String())
	<-ctx.Done()
	cfg.Logger.Print("utc-milter shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := filter.Shutdown(shutdownCtx); err != nil {
		filter.Close()
		return err
	}
	filter.Wait()
	return nil
}

func (cfg Config) withDefaults() Config {
	if cfg.Network == "" {
		cfg.Network = "unix"
	}
	if cfg.Address == "" {
		cfg.Address = "/run/utc-milter/utc-milter.sock"
	}
	if cfg.SocketMode == 0 {
		cfg.SocketMode = 0o660
	}
	if cfg.ShutdownTimeout == 0 {
		cfg.ShutdownTimeout = 10 * time.Second
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Logger == nil {
		cfg.Logger = log.New(os.Stdout, "", log.LstdFlags)
	}
	return cfg
}

func utcDateDecision(now func() time.Time) mailfilter.DecisionModificationFunc {
	return func(_ context.Context, trx mailfilter.Trx) (mailfilter.Decision, error) {
		headers := trx.Headers()
		if headers.Value("Date") == "" {
			headers.SetDate(now().UTC())
			return mailfilter.Accept, nil
		}

		date, err := headers.Date()
		if err != nil {
			headers.SetDate(now().UTC())
			return mailfilter.Accept, nil
		}
		headers.SetDate(date.UTC())
		return mailfilter.Accept, nil
	}
}

func prepareUnixSocket(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create socket directory: %w", err)
	}
	info, err := os.Lstat(path)
	if err == nil {
		if info.Mode()&os.ModeSocket == 0 {
			return fmt.Errorf("%s exists and is not a socket", path)
		}
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("remove stale socket: %w", err)
		}
		return nil
	}
	if os.IsNotExist(err) {
		return nil
	}
	return fmt.Errorf("stat socket: %w", err)
}

func setSocketAccess(path string, mode os.FileMode, ownerName, groupName string) error {
	if err := os.Chmod(path, mode); err != nil {
		return fmt.Errorf("chmod socket: %w", err)
	}

	uid := -1
	gid := -1
	var err error
	if ownerName != "" {
		uid, err = lookupUID(ownerName)
		if err != nil {
			return err
		}
	}
	if groupName != "" {
		gid, err = lookupGID(groupName)
		if err != nil {
			return err
		}
	}
	if uid != -1 || gid != -1 {
		if err := os.Chown(path, uid, gid); err != nil {
			return fmt.Errorf("chown socket: %w", err)
		}
	}
	return nil
}

func lookupUID(name string) (int, error) {
	if uid, err := strconv.Atoi(name); err == nil {
		return uid, nil
	}
	u, err := user.Lookup(name)
	if err != nil {
		return 0, fmt.Errorf("lookup user %q: %w", name, err)
	}
	return strconv.Atoi(u.Uid)
}

func lookupGID(name string) (int, error) {
	if gid, err := strconv.Atoi(name); err == nil {
		return gid, nil
	}
	g, err := user.LookupGroup(name)
	if err != nil {
		return 0, fmt.Errorf("lookup group %q: %w", name, err)
	}
	return strconv.Atoi(g.Gid)
}
