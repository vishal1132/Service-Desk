package config

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
)

// R is the redis-specific options
type R struct {
	// Addr is the Redis host and port to connect to
	Addr string

	// User is the Redis user
	User string

	// Password is the Redis password
	Password string

	// Insecure is whether we should connect to Redis over plain text
	Insecure bool

	// SkipVerify is whether we skip x.509 certification validation
	SkipVerify bool
}

// D is the database struct
type D struct {
}

func secureRedisCredentials(s string, insecure bool) (host, user, password string, err error) {
	u, err := url.Parse(s)
	if err != nil {
		return "", "", "", err
	}

	switch u.Scheme {
	case "rediss":
		pass, _ := u.User.Password()
		return u.Host, u.User.Username(), pass, nil

	case "redis":
		h, p, err := net.SplitHostPort(u.Host)
		if err != nil {
			if !strings.Contains(err.Error(), "missing port in address") {
				return "", "", "", err
			}

			h = u.Host
		}

		if p == "" {
			p = "6379"
		}

		pi, err := strconv.Atoi(p)
		if err != nil {
			return "", "", "", err
		}

		if !insecure { // it's secure
			pi++
		}

		pass, _ := u.User.Password()

		return net.JoinHostPort(h, strconv.Itoa(pi)), u.User.Username(), pass, nil

	default:
		return "", "", "", fmt.Errorf("unknown scheme: %s", u.Scheme)
	}
}

// C is the configuration struct.
type C struct {
	// LogLevel is the logging level
	// Env: LOG_LEVEL
	LogLevel zerolog.Level

	// Port is the TCP port for web workers to listen on, loaded from PORT
	// Env: PORT
	Port uint16

	// Redis is the Redis configuration
	// Env: REDIS_URL
	Redis R

	// D is the database configuration
	Database D
}

// LoadEnv loads the configuration from the appropriate environment variables.
func LoadEnv() (C, error) {
	godotenv.Load()
	var c C

	if p := os.Getenv("PORT"); len(p) > 0 {
		u, err := strconv.ParseUint(p, 10, 16)
		if err != nil {
			return C{}, fmt.Errorf("failed to parse PORT: %w", err)
		}

		c.Port = uint16(u)
	}

	if r := os.Getenv("REDIS_URL"); len(r) > 0 {
		c.Redis.Insecure = os.Getenv("REDIS_INSECURE") == "1"
		c.Redis.SkipVerify = os.Getenv("REDIS_SKIPVERIFY") == "1"

		a, u, p, err := secureRedisCredentials(r, c.Redis.Insecure)
		if err != nil {
			return C{}, fmt.Errorf("failed to parse REDIS_URL: %w", err)
		}

		c.Redis.Addr = a
		c.Redis.User = u
		c.Redis.Password = p
	}

	ll := os.Getenv("LOG_LEVEL")
	if len(ll) == 0 {
		ll = "info"
	}

	l, err := zerolog.ParseLevel(ll)
	if err != nil {
		return C{}, fmt.Errorf("failed to parse LOG_LEVEL: %w", err)
	}

	c.LogLevel = l

	return c, nil
}

// DefaultLogger returns a zerolog.Logger using settings from our config struct.
func DefaultLogger(cfg C) zerolog.Logger {
	// set up zerolog
	zerolog.TimestampFieldName = "timestamp"
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	zerolog.SetGlobalLevel(cfg.LogLevel)

	// set up logging
	return zerolog.New(os.Stdout).
		With().Timestamp().Logger()
}

// DefaultRedis returns a default Redis config from our own config struct.
func DefaultRedis(cfg C) *redis.Options {
	r := &redis.Options{
		Network:      "tcp",
		Addr:         cfg.Redis.Addr,
		Password:     cfg.Redis.Password,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
		PoolSize:     20,
		MinIdleConns: 5,
		PoolTimeout:  2 * time.Second,
	}

	// if Redis is TLS secured
	if !cfg.Redis.Insecure {
		r.TLSConfig = &tls.Config{
			InsecureSkipVerify: cfg.Redis.SkipVerify,
		} // #nosec G402 -- Heroku Redis has an untrusted cert
	}

	return r
}
